FROM nginx:1.25-alpine
COPY deploy/fullstack/nginx.conf /etc/nginx/conf.d/default.conf
