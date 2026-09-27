// Independent TypeScript check of the golden vectors (gate G1).
// EIP-712 via ethers v5 (what the frontend signs with), Borsh via the `borsh` package NEAR uses,
// Ed25519 via @near-js/crypto. Nothing here imports the Go generator's logic.
import { readFileSync } from 'node:fs';
import { ethers } from 'ethers';
import { serialize } from 'borsh';
import { PublicKey, KeyPairEd25519 } from '@near-js/crypto';
import { createTransaction, actionCreators, encodeTransaction, SignedTransaction, Signature } from '@near-js/transactions';
import bs58 from 'bs58';

const dir = new URL('..', import.meta.url).pathname;
const load = (f: string) => JSON.parse(readFileSync(dir + f, 'utf8'));

let failures = 0;
let checks = 0;
const eq = (label: string, got: string | number | boolean, want: string | number | boolean) => {
    checks++;
    const g = typeof got === 'string' ? got.toLowerCase() : got;
    const w = typeof want === 'string' ? want.toLowerCase() : want;
    if (g !== w) {
        failures++;
        console.error(`FAIL ${label}\n  got  ${got}\n  want ${want}`);
    }
};

const { keccak256, toUtf8Bytes, hexlify, arrayify, hexZeroPad, sha256, recoverAddress } = ethers.utils;

// ---- addr20 ----
const addr = load('addr20.json');
const toBE6 = (n: number) => hexZeroPad(ethers.BigNumber.from(n).toHexString(), 6).slice(2);
for (const c of addr.valid) {
    const a20 = '0x' + keccak256(toUtf8Bytes('near:' + c.accountId)).slice(26);
    eq(`addr20 ${c.accountId}`, a20, c.addr20);
    eq(`checksum ${c.accountId}`, ethers.utils.getAddress(a20), c.addr20);
    eq(`sub32 ${c.accountId}`, '0x' + toBE6(c.brokerId) + a20.slice(2) + toBE6(c.subaccountNumber), c.subaccountBytes32);
}

// ---- EIP-712 (frontend path) ----
const orderTypes = {
    Order: [
        { name: 'subAccountId', type: 'bytes32' },
        { name: 'priceX18', type: 'int128' },
        { name: 'amount', type: 'int128' },
        { name: 'expiration', type: 'uint64' },
        { name: 'isReduce', type: 'bool' },
        { name: 'sessionKey', type: 'address' },
        { name: 'chainId', type: 'uint256' },
        { name: 'productId', type: 'uint32' },
    ],
};
const eip = load('eip712.json');
for (const c of eip.order) {
    const vc = '0x' + keccak256(toUtf8Bytes(c.domain.contractAccount)).slice(26);
    eq(`verifyingContract ${c.domain.contractAccount}`, vc, c.domain.verifyingContract);
    const domain = { name: c.domain.name, version: c.domain.version, chainId: c.domain.chainId, verifyingContract: vc };
    const msg = { ...c.message, isReduce: c.message.isReduce === 'true' };
    eq(`domainSeparator ${c.domain.chainId}`, ethers.utils._TypedDataEncoder.hashDomain(domain), c.domain.domainSeparator);
    eq(`typeHash`, keccak256(toUtf8Bytes(ethers.utils._TypedDataEncoder.from(orderTypes).encodeType('Order'))), c.typeHash);
    eq(`structHash ${c.domain.chainId}`, ethers.utils._TypedDataEncoder.from(orderTypes).hash(msg), c.structHash);
    const digest = ethers.utils._TypedDataEncoder.hash(domain, orderTypes, msg);
    eq(`digest ${c.domain.chainId}`, digest, c.digest);
    const wallet = new ethers.Wallet(c.signerPrivateKeyTestOnly);
    eq(`signer`, wallet.address, c.signer);
    eq(`recover ${c.domain.chainId}`, recoverAddress(digest, c.signature), c.signer);
    // ethers signs with RFC6979 like go-ethereum, so the signature itself must be identical
    eq(`signature ${c.domain.chainId}`, ethers.utils.joinSignature(wallet._signingKey().signDigest(digest)), c.signature);
}

// ---- NEP-413 (wallet path) ----
const nep = load('nep413.json');
const nepSchema = {
    struct: {
        message: 'string',
        nonce: { array: { type: 'u8', len: 32 } },
        recipient: 'string',
        callbackUrl: { option: 'string' },
    },
};
const pk = PublicKey.fromString(nep.publicKey);
for (const c of nep.cases) {
    const tag = serialize('u32', nep.tag);
    const payload = serialize(nepSchema, {
        message: c.message,
        nonce: Array.from(arrayify(c.nonce)),
        recipient: c.recipient,
        callbackUrl: c.callbackUrl ?? null,
    });
    const bytes = new Uint8Array([...tag, ...payload]);
    eq(`nep413 borsh ${c.recipient}`, hexlify(bytes), c.borshHex);
    eq(`nep413 nonce b64 ${c.recipient}`, Buffer.from(arrayify(c.nonce)).toString('base64'), c.nonceBase64);
    const hash = sha256(bytes);
    eq(`nep413 sha256 ${c.recipient}`, hash, c.sha256);
    eq(`nep413 sig ${c.recipient}`, pk.verify(arrayify(hash), Buffer.from(c.signatureBase64, 'base64')), true);
    const tampered = sha256(serialize(nepSchema, { message: c.message, nonce: Array.from(arrayify(c.nonce)), recipient: 'evil.near', callbackUrl: c.callbackUrl ?? null }));
    eq(`nep413 tamper rejected ${c.recipient}`, pk.verify(arrayify(tampered), Buffer.from(c.signatureBase64, 'base64')), false);
}

// ---- Borsh order payload ----
const b = load('borsh.json').order;
const orderSchema = {
    struct: {
        subaccount: { array: { type: 'u8', len: 32 } },
        price_x18: 'i128',
        amount: 'i128',
        expiration: 'u64',
        is_reduce: 'bool',
        session_key: { array: { type: 'u8', len: 20 } },
        product_id: 'u32',
    },
};
const enc = serialize(orderSchema, {
    subaccount: Array.from(arrayify(b.subaccount)),
    price_x18: BigInt(b.priceX18),
    amount: BigInt(b.amount),
    expiration: BigInt(b.expiration),
    is_reduce: b.isReduce,
    session_key: Array.from(arrayify('0x' + b.sessionKey)),
    product_id: b.productId,
});
eq('borsh order', hexlify(enc), b.borshHex);
eq('borsh envelope', hexlify(new Uint8Array([b.typeByte, ...enc])), b.envelopeHex);

// ---- Signed NEAR transaction (Go nearchain vs @near-js/transactions) ----
const nt = load('neartx.json');
const seed = arrayify(nt.privateSeedTestOnly);
const pkBytes = PublicKey.fromString(nt.publicKey).data;
const kp = new KeyPairEd25519(bs58.encode(new Uint8Array([...seed, ...pkBytes])));
eq('neartx pubkey', kp.getPublicKey().toString(), nt.publicKey);
const ntx = createTransaction(
    nt.signerId,
    kp.getPublicKey(),
    nt.receiverId,
    BigInt(nt.nonce),
    nt.actions.map((a: any) => actionCreators.functionCall(a.methodName, Buffer.from(a.argsJson), BigInt(a.gas), BigInt(a.deposit))),
    bs58.decode(nt.blockHash),
);
const txBytes = encodeTransaction(ntx);
eq('neartx borsh', hexlify(txBytes), nt.txBorshHex);
const txHash = arrayify(sha256(txBytes));
eq('neartx hash', bs58.encode(txHash), nt.txHash);
const sig = kp.sign(txHash);
const stx = new SignedTransaction({ transaction: ntx, signature: new Signature({ keyType: 0, data: sig.signature }) });
eq('neartx signed', Buffer.from(stx.encode()).toString('base64'), nt.signedBase64);

// ---- near-stocks session-key messages (what the frontend will sign with ethers) ----
const ns = load('nearsign.json');
const nsDomain = {
    name: 'near-stocks', version: '1', chainId: ns.chainId,
    verifyingContract: '0x' + keccak256(toUtf8Bytes(ns.contractAccount)).slice(26),
};
const registerTypes = { Register: [
    { name: 'subAccountId', type: 'bytes32' }, { name: 'userAddress', type: 'address' },
    { name: 'sessionKey', type: 'address' }, { name: 'expiryTimeStamp', type: 'uint128' },
    { name: 'nonce', type: 'uint128' }, { name: 'chainId', type: 'uint256' },
] };
const withdrawTypes = { NearWithdraw: [
    { name: 'subAccountId', type: 'bytes32' }, { name: 'sessionKey', type: 'address' },
    { name: 'productId', type: 'uint32' }, { name: 'amount', type: 'uint128' },
    { name: 'nonce', type: 'uint128' }, { name: 'receiver', type: 'string' },
    { name: 'chainId', type: 'uint256' },
] };
const sessionWallet = new ethers.Wallet(eip.order[0].signerPrivateKeyTestOnly);
eq('nearsign session key', sessionWallet.address, ns.sessionKey);
for (const [label, types, msg] of [['register', registerTypes, ns.register], ['withdraw', withdrawTypes, ns.withdraw]] as const) {
    const { digest, signature, ...fields } = msg as any;
    const d = ethers.utils._TypedDataEncoder.hash(nsDomain, types as any, fields);
    eq(`nearsign ${label} digest`, d, digest);
    eq(`nearsign ${label} recover`, recoverAddress(d, signature), ns.sessionKey);
    eq(`nearsign ${label} signature`, await sessionWallet._signTypedData(nsDomain, types as any, fields), signature);
}

// requests.json: every session-key request kept on NEAR, as the frontend (ethers) signs it
const rq = load('requests.json');
const rqDomain = { name: 'near-stocks', version: '1', chainId: rq.chainId, verifyingContract: nsDomain.verifyingContract };
for (const r of rq.requests as any[]) {
    const msg: Record<string, any> = {};
    for (const f of r.types) {
        const k = f.name;
        if (k === 'subAccountId') msg[k] = rq.subAccountId;
        else if (k === 'sessionKey') msg[k] = rq.sessionKey;
        else if (k === 'chainId') msg[k] = rq.chainId;
        else if (k === 'tokenAmount' || (k === 'amount' && r.amount !== undefined)) msg[k] = r.amount;
        else msg[k] = r[k];
    }
    const types = { [r.primaryType]: r.types };
    const d = ethers.utils._TypedDataEncoder.hash(rqDomain, types, msg);
    eq(`request ${r.primaryType} digest`, d, r.digest);
    eq(`request ${r.primaryType} signature`, await sessionWallet._signTypedData(rqDomain, types, msg), r.signature);
}

console.log(`${checks - failures}/${checks} checks passed`);
process.exit(failures ? 1 : 0);
