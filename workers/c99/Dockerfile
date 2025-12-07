FROM gcc:12

RUN apt-get update && apt-get install -y python3 python3-pip python3-venv git cmake

# Create virtual environment and install redis
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install redis

# Install cJSON library to /usr prefix
RUN git clone https://github.com/DaveGamble/cJSON.git /tmp/cJSON && \
    cd /tmp/cJSON && \
    mkdir build && cd build && \
    cmake -DCMAKE_INSTALL_PREFIX=/usr .. && make && make install && \
    ldconfig && \
    rm -rf /tmp/cJSON

WORKDIR /app

RUN mkdir -p /tmp/sandbox

COPY . .

CMD ["python", "worker.py"]
