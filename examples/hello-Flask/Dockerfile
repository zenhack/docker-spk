FROM zenhack/sandstorm-http-bridge:276 as builder

RUN apk add \
	python3 \
	build-base \
	python3-dev \
	py3-virtualenv

WORKDIR /app

RUN virtualenv .venv
RUN .venv/bin/pip install gunicorn
ADD * ./
RUN .venv/bin/pip install .

FROM zenhack/sandstorm-http-bridge:276
RUN apk add python3
COPY --from=builder /app /app
