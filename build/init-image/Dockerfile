FROM  ghcr.io/dynamia-ai/init-base:v0.0.1

WORKDIR /app

COPY ./s6 /app/s6
COPY ./hack/* /app/

RUN mkdir -p /app/code-server && cp -r /usr/local/code-server/* /app/code-server/

ENTRYPOINT ["ls"]
