FROM silkeh/clang:15

RUN apt-get update && apt-get install -y python3 python3-pip python3-venv curl

# Create virtual environment and install redis
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install redis

# Install nlohmann/json header-only library to multiple include paths
RUN mkdir -p /usr/include/nlohmann /usr/local/include/nlohmann && \
    curl -L https://github.com/nlohmann/json/releases/download/v3.11.2/json.hpp \
    -o /usr/include/nlohmann/json.hpp && \
    cp /usr/include/nlohmann/json.hpp /usr/local/include/nlohmann/json.hpp

WORKDIR /app

RUN mkdir -p /tmp/sandbox

COPY . .

CMD ["python", "worker.py"]
