FROM zenhack/sandstorm-http-bridge:238 as builder

RUN apk add \
	python2 \
	build-base \
	python2-dev \
	py2-virtualenv

WORKDIR /app

RUN virtualenv .venv
RUN .venv/bin/pip install gunicorn
ADD * ./
RUN .venv/bin/pip install .

FROM zenhack/sandstorm-http-bridge:238
RUN apk add python2
COPY --from=builder /app /app
