"""
리소스 제한 모듈 - Lambda 환경 시뮬레이션
- CPU 시간 제한
- 메모리 제한
- 프로세스/스레드 수 제한
"""

import resource
import os


def set_limits():
    """
    프로세스에 리소스 제한 적용
    환경변수로 설정 가능: CPU_TIME_LIMIT, MEMORY_LIMIT_MB
    """
    cpu_time_limit = int(os.environ.get('CPU_TIME_LIMIT', 30))  # 초
    memory_limit_mb = int(os.environ.get('MEMORY_LIMIT_MB', 512))  # MB

    try:
        # 1. CPU 시간 제한 (소프트, 하드)
        resource.setrlimit(resource.RLIMIT_CPU, (cpu_time_limit, cpu_time_limit + 5))
        print(f"[Limiter] CPU time limit: {cpu_time_limit}s")

        # 2. 메모리 제한 (Address Space)
        memory_bytes = memory_limit_mb * 1024 * 1024
        resource.setrlimit(resource.RLIMIT_AS, (memory_bytes, memory_bytes))
        print(f"[Limiter] Memory limit: {memory_limit_mb}MB")

        # 3. 파일 디스크립터 제한
        resource.setrlimit(resource.RLIMIT_NOFILE, (256, 256))

        # 4. 프로세스 수 제한 (fork bomb 방지)
        resource.setrlimit(resource.RLIMIT_NPROC, (50, 50))

        # 5. 파일 크기 제한 (10MB)
        max_file_size = 10 * 1024 * 1024
        resource.setrlimit(resource.RLIMIT_FSIZE, (max_file_size, max_file_size))

    except ValueError as e:
        print(f"[Limiter] Warning: Could not set some limits: {e}")
    except resource.error as e:
        print(f"[Limiter] Warning: Resource limit error: {e}")


def get_current_usage():
    """현재 리소스 사용량 확인"""
    usage = resource.getrusage(resource.RUSAGE_SELF)
    return {
        'cpu_time': usage.ru_utime + usage.ru_stime,  # 초
        'max_rss': usage.ru_maxrss / 1024,  # MB (Linux는 KB 단위)
    }
