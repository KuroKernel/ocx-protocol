#define _GNU_SOURCE
#include <dlfcn.h>
#include <stdint.h>
#include <time.h>
#include <sys/random.h>
#include <string.h>
#include <stdlib.h>

// Deterministic RNG state
static uint64_t s = 0;

// SplitMix64 PRNG for deterministic random number generation
static inline uint64_t splitmix64() {
    s += 0x9e3779b97f4a7c15ULL;
    uint64_t z = s;
    z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9ULL;
    z = (z ^ (z >> 27)) * 0x94d049bb133111ebULL;
    return z ^ (z >> 31);
}

// Initialize the deterministic RNG with seed from environment
__attribute__((constructor))
static void init() {
    const char* env = getenv("OCX_SEED");
    if (env) {
        s = strtoull(env, NULL, 16);
    } else {
        s = 0xdeadbeefUL; // Default seed
    }
}

// Override getrandom to use deterministic RNG
ssize_t getrandom(void *buf, size_t buflen, unsigned int flags) {
    (void)flags; // Ignore flags for deterministic behavior
    
    uint8_t *p = (uint8_t*)buf;
    for (size_t i = 0; i < buflen; i += 8) {
        uint64_t r = splitmix64();
        size_t m = (buflen - i < 8) ? (buflen - i) : 8;
        memcpy(p + i, &r, m);
    }
    return buflen;
}

// Override clock_gettime to return fixed timestamps
int clock_gettime(clockid_t clk_id, struct timespec *tp) {
    if (clk_id == CLOCK_REALTIME || clk_id == CLOCK_MONOTONIC) {
        // Return fixed timestamp for deterministic behavior
        tp->tv_sec = 1;
        tp->tv_nsec = 0;
        return 0;
    }
    
    // For other clock types, use the real function
    static int (*real_fn)(clockid_t, struct timespec*) = NULL;
    if (!real_fn) {
        real_fn = dlsym(RTLD_NEXT, "clock_gettime");
    }
    return real_fn ? real_fn(clk_id, tp) : -1;
}

// Override gettimeofday for additional time control
int gettimeofday(struct timeval *tv, struct timezone *tz) {
    if (tv) {
        tv->tv_sec = 1;
        tv->tv_usec = 0;
    }
    if (tz) {
        tz->tz_minuteswest = 0;
        tz->tz_dsttime = 0;
    }
    return 0;
}

// Override time function
time_t time(time_t *tloc) {
    time_t fixed_time = 1;
    if (tloc) {
        *tloc = fixed_time;
    }
    return fixed_time;
}
