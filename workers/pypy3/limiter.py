import resource
import os

def set_limits():
    """Apply resource limits to the process (simulating Lambda environment)"""
    cpu_time_limit = int(os.environ.get('CPU_TIME_LIMIT', 30))  # seconds
    memory_limit_mb = int(os.environ.get('MEMORY_LIMIT_MB', 512))  # MB

    try:
        # CPU time limit (soft, hard)
        resource.setrlimit(resource.RLIMIT_CPU, (cpu_time_limit, cpu_time_limit + 5))
        print(f"[Limiter] CPU time limit: {cpu_time_limit}s")

        # Memory limit (Address Space)
        memory_bytes = memory_limit_mb * 1024 * 1024
        resource.setrlimit(resource.RLIMIT_AS, (memory_bytes, memory_bytes))
        print(f"[Limiter] Memory limit: {memory_limit_mb}MB")

        # File descriptor limit
        resource.setrlimit(resource.RLIMIT_NOFILE, (256, 256))

        # Process limit (prevent fork bombs)
        resource.setrlimit(resource.RLIMIT_NPROC, (50, 50))

        # File size limit (10MB)
        max_file_size = 10 * 1024 * 1024
        resource.setrlimit(resource.RLIMIT_FSIZE, (max_file_size, max_file_size))

    except ValueError as e:
        print(f"[Limiter] Warning: Could not set some limits: {e}")
