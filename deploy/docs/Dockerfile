# The guide's image: the site task docs:build leaves in public, served by nginx.
# Build from the module root: docker build -f deploy/docs/Dockerfile .
FROM nginxinc/nginx-unprivileged:1.29-alpine

COPY deploy/docs/nginx.conf /etc/nginx/conf.d/default.conf
COPY public /usr/share/nginx/html/ember

EXPOSE 8080
