FROM ruby:3.2-slim

RUN apt-get update && apt-get install -y build-essential

WORKDIR /app

COPY Gemfile ./
RUN bundle install

COPY . .

CMD ["ruby", "worker.rb"]
