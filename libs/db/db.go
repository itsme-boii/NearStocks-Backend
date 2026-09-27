package db

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func Init() {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		log.Fatal("DSN env variable is not set")
	}

	var err error

	xlog.Infof("Database - Connecting with database....")
	if os.Getenv("DB_DEBUG") == "1" {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	} else {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}

	if err != nil {
		panic("database - failed to connect database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		xlog.Errorf("Database - Failed to edit config")

	}

	sqlDB.SetMaxOpenConns(400)
	sqlDB.SetMaxIdleConns(80)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	fmt.Println("Database Connected. Connection pooling configured. Auto migrating....")

	err = db.AutoMigrate(&SubaccountTable{}, &AuthTable{}, &SigningKeyTable{}, &BrokerTable{}, &MarketTable{}, &OrderTable{}, &FillTable{}, &IpTable{}, &RewardTable{}, &PointsTable{}, &FillOrderTable{}, &ReferralUserTable{}, &FundingRateTable{}, &PnlTable{}, &BatchTable{}, &StakingTable{}, &LiquidationTable{}, &LogxTokenUserTable{}, &TokenVestingTable{}, &UserFeeRewardsTable{}, &SubaccountLastDepositTable{}, &ReferralRewardTable{}, &ReferralHistoryTable{}, &LotteryFlowTable{}, &OptionsTable{}, &PreMarketTable{}, &PreMarketUserTable{}, &PreMarketCandleTable{}, &SyntheticSpotUserTable{}, &ProposalTable{}, &VoteTable{}, &AffiliateTable{}, &AirdropAllocationTable{}, &CantonPartyTable{}, &DepositWithdrawTable{}, &NearAccountTable{}, &IntentsTransferTable{}, &NearOrderDigestTable{}, &NearChainEventTable{}, &NearReconcileDiffTable{})

	//Please do not add desposit_withdraw_tabel here
	if err != nil {
		panic("database - failed to migrate database")
	}
}

// ExecSQL executes raw SQL query. Used by tests.
func ExecSQL(query string, args ...interface{}) error {
	return db.Exec(query, args...).Error
}
