FROM node:24-alpine as build

WORKDIR /app
COPY . /app
RUN npm run build

FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
ENTRYPOINT ["/bin/sh", "-c", "envsubst < /usr/share/nginx/html/assets/config.json.template > /usr/share/nginx/html/assets/config.json && exec nginx -g 'daemon off;'"]