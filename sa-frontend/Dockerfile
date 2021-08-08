### build stage ###
# pull node as builder base image
FROM node:10 AS builder

WORKDIR /app

# --build-arg
# the URL here is the public URI of the sa webapp, not the cluster internal one
ARG SA_WEBAPP_URL=http://localhost
ARG SA_WEBAPP_PORT=57068

ENV REACT_APP_BE_SERVICE_URL=$SA_WEBAPP_URL
ENV REACT_APP_BE_SERVICE_PORT=$SA_WEBAPP_PORT

COPY package.json ./

RUN npm install

COPY . ./

RUN npm run build

### run stage ###
FROM nginx:1.19.0

WORKDIR /usr/share/nginx/html

RUN rm -rf ./*

COPY --from=builder /app/build .

ENTRYPOINT ["nginx", "-g", "daemon off;"]
