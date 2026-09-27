# Development stage
FROM --platform=linux/amd64 public.ecr.aws/l8t0j4y9/node-21:latest AS development

WORKDIR /usr/src/app

COPY package*.json ./

RUN npm install --only=development

COPY . .

RUN npm run build

# Production stage
FROM --platform=linux/amd64  public.ecr.aws/l8t0j4y9/node-21:latest as production

ARG NODE_ENV=production
ENV NODE_ENV=${NODE_ENV}

WORKDIR /usr/src/app

COPY package*.json ./

RUN npm install --only=production

COPY . .

COPY --from=development /usr/src/app/dist ./dist

CMD ["node", "dist/main"]