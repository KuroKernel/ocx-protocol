;; OCX Reputation Aggregator - Deterministic Multi-Platform Score Computation
;; Designed for OCX D-MVM execution with cryptographic receipt generation
;; Gas target: 238 units | Execution time: <5ms | Determinism: 100%

(module
  ;; ===================================================================
  ;; IMPORTS FROM OCX RUNTIME (provided by pkg/deterministicvm/)
  ;; ===================================================================

  ;; Fetch platform data deterministically via OCX runtime
  ;; param: platform_id (0=GitHub, 1=LinkedIn, 2=Uber)
  ;; param: data_ptr (pointer to user_id string in linear memory)
  ;; returns: pointer to normalized score data
  (import "ocx" "fetch_data" (func $fetch_data (param i32 i32) (result i32)))

  ;; Get deterministic timestamp from ChaCha20/12 PRNG
  ;; returns: unix timestamp (seconds since epoch)
  (import "ocx" "get_timestamp" (func $get_timestamp (result i64)))

  ;; SHA-256 hash computation (for receipt generation)
  ;; param: data_ptr, data_len
  ;; returns: pointer to 32-byte hash
  (import "ocx" "hash_sha256" (func $hash_sha256 (param i32 i32) (result i32)))

  ;; Debug logging (only enabled in test mode)
  ;; param: msg_ptr, msg_len
  (import "ocx" "log_debug" (func $log_debug (param i32 i32)))

  ;; ===================================================================
  ;; LINEAR MEMORY LAYOUT
  ;; ===================================================================
  ;; 0x0000-0x00FF: Input buffer (256 bytes - user_id, platform flags)
  ;; 0x0100-0x01FF: Platform scores (GitHub, LinkedIn, Uber - 24 bytes)
  ;; 0x0200-0x02FF: Intermediate computation buffer
  ;; 0x0300-0x03FF: Output buffer (aggregated score + metadata)
  ;; 0x0400-0x07FF: Working memory for string operations

  (memory (export "memory") 1)  ;; 64KB (1 page), enough for our needs

  ;; ===================================================================
  ;; GLOBAL CONSTANTS
  ;; ===================================================================

  ;; Platform weights (IEEE 754 double precision)
  ;; These are carefully chosen to avoid floating-point rounding issues
  (global $WEIGHT_GITHUB f64 (f64.const 0.40))      ;; 40% - code contributions
  (global $WEIGHT_LINKEDIN f64 (f64.const 0.35))    ;; 35% - professional network
  (global $WEIGHT_UBER f64 (f64.const 0.25))        ;; 25% - service quality

  ;; Score scaling factor (0-100 input → 0-1000 output)
  (global $SCALE_FACTOR f64 (f64.const 10.0))

  ;; Platform IDs
  (global $PLATFORM_GITHUB i32 (i32.const 0))
  (global $PLATFORM_LINKEDIN i32 (i32.const 1))
  (global $PLATFORM_UBER i32 (i32.const 2))

  ;; Memory offsets
  (global $INPUT_BUFFER i32 (i32.const 0x0000))
  (global $SCORES_BUFFER i32 (i32.const 0x0100))
  (global $COMPUTE_BUFFER i32 (i32.const 0x0200))
  (global $OUTPUT_BUFFER i32 (i32.const 0x0300))

  ;; ===================================================================
  ;; PRIMARY ENTRY POINT
  ;; ===================================================================

  ;; Compute aggregated reputation score from multiple platforms
  ;; param $user_id: User identifier (offset in linear memory)
  ;; param $user_id_len: Length of user_id string
  ;; param $platform_flags: Bitmask indicating which platforms to include
  ;;   bit 0 = GitHub, bit 1 = LinkedIn, bit 2 = Uber
  ;; returns: Aggregated score (f64, range 0-1000)
  (func $compute_reputation (param $user_id i32) (param $user_id_len i32) (param $platform_flags i32) (result f64)
    (local $github_score f64)
    (local $linkedin_score f64)
    (local $uber_score f64)
    (local $weighted_sum f64)
    (local $total_weight f64)
    (local $final_score f64)
    (local $platform_count i32)

    ;; Initialize
    (local.set $weighted_sum (f64.const 0.0))
    (local.set $total_weight (f64.const 0.0))
    (local.set $platform_count (i32.const 0))

    ;; ---------------------------------------------------------------
    ;; GITHUB SCORE FETCHING
    ;; ---------------------------------------------------------------
    (if (i32.and (local.get $platform_flags) (i32.const 0x01))
      (then
        ;; Fetch GitHub score via OCX runtime
        (local.set $github_score
          (call $fetch_and_normalize
            (global.get $PLATFORM_GITHUB)
            (local.get $user_id)
            (local.get $user_id_len)))

        ;; Add to weighted sum
        (local.set $weighted_sum
          (f64.add
            (local.get $weighted_sum)
            (f64.mul (local.get $github_score) (global.get $WEIGHT_GITHUB))))

        ;; Accumulate weight
        (local.set $total_weight
          (f64.add (local.get $total_weight) (global.get $WEIGHT_GITHUB)))

        ;; Increment platform count
        (local.set $platform_count (i32.add (local.get $platform_count) (i32.const 1)))

        ;; Log debug info (only in test mode)
        (call $log_score_debug
          (i32.const 0)  ;; Platform: GitHub
          (local.get $github_score))
      )
    )

    ;; ---------------------------------------------------------------
    ;; LINKEDIN SCORE FETCHING
    ;; ---------------------------------------------------------------
    (if (i32.and (local.get $platform_flags) (i32.const 0x02))
      (then
        ;; Fetch LinkedIn score via OCX runtime
        (local.set $linkedin_score
          (call $fetch_and_normalize
            (global.get $PLATFORM_LINKEDIN)
            (local.get $user_id)
            (local.get $user_id_len)))

        ;; Add to weighted sum
        (local.set $weighted_sum
          (f64.add
            (local.get $weighted_sum)
            (f64.mul (local.get $linkedin_score) (global.get $WEIGHT_LINKEDIN))))

        ;; Accumulate weight
        (local.set $total_weight
          (f64.add (local.get $total_weight) (global.get $WEIGHT_LINKEDIN)))

        ;; Increment platform count
        (local.set $platform_count (i32.add (local.get $platform_count) (i32.const 1)))

        ;; Log debug info
        (call $log_score_debug
          (i32.const 1)  ;; Platform: LinkedIn
          (local.get $linkedin_score))
      )
    )

    ;; ---------------------------------------------------------------
    ;; UBER SCORE FETCHING
    ;; ---------------------------------------------------------------
    (if (i32.and (local.get $platform_flags) (i32.const 0x04))
      (then
        ;; Fetch Uber score via OCX runtime
        (local.set $uber_score
          (call $fetch_and_normalize
            (global.get $PLATFORM_UBER)
            (local.get $user_id)
            (local.get $user_id_len)))

        ;; Add to weighted sum
        (local.set $weighted_sum
          (f64.add
            (local.get $weighted_sum)
            (f64.mul (local.get $uber_score) (global.get $WEIGHT_UBER))))

        ;; Accumulate weight
        (local.set $total_weight
          (f64.add (local.get $total_weight) (global.get $WEIGHT_UBER)))

        ;; Increment platform count
        (local.set $platform_count (i32.add (local.get $platform_count) (i32.const 1)))

        ;; Log debug info
        (call $log_score_debug
          (i32.const 2)  ;; Platform: Uber
          (local.get $uber_score))
      )
    )

    ;; ---------------------------------------------------------------
    ;; SCORE AGGREGATION AND SCALING
    ;; ---------------------------------------------------------------

    ;; Check if at least one platform was processed
    (if (i32.eq (local.get $platform_count) (i32.const 0))
      (then
        ;; No platforms enabled - return 0
        (return (f64.const 0.0))
      )
    )

    ;; Normalize by total weight (handles partial platform sets)
    (local.set $final_score
      (f64.div (local.get $weighted_sum) (local.get $total_weight)))

    ;; Scale from 0-100 range to 0-1000 range
    (local.set $final_score
      (f64.mul (local.get $final_score) (global.get $SCALE_FACTOR)))

    ;; Clamp to valid range [0, 1000]
    (local.set $final_score
      (call $clamp_score (local.get $final_score)))

    ;; Store result in output buffer for receipt generation
    (f64.store (global.get $OUTPUT_BUFFER) (local.get $final_score))

    ;; Return final score
    (local.get $final_score)
  )

  ;; ===================================================================
  ;; HELPER FUNCTIONS
  ;; ===================================================================

  ;; Fetch platform score and normalize to 0-100 range
  ;; This is a wrapper around the OCX runtime fetch_data import
  ;; param $platform_id: Platform identifier (0=GitHub, 1=LinkedIn, 2=Uber)
  ;; param $user_id_ptr: Pointer to user ID string
  ;; param $user_id_len: Length of user ID string
  ;; returns: Normalized score (f64, range 0-100)
  (func $fetch_and_normalize (param $platform_id i32) (param $user_id_ptr i32) (param $user_id_len i32) (result f64)
    (local $data_ptr i32)
    (local $score f64)

    ;; Call OCX runtime to fetch data
    ;; The runtime will handle OAuth, caching, rate limiting, etc.
    ;; Returns pointer to 8-byte IEEE 754 double in linear memory
    (local.set $data_ptr
      (call $fetch_data (local.get $platform_id) (local.get $user_id_ptr)))

    ;; Load the normalized score from memory
    (local.set $score (f64.load (local.get $data_ptr)))

    ;; Clamp to valid range [0, 100]
    (local.set $score (call $clamp_score_100 (local.get $score)))

    (local.get $score)
  )

  ;; Clamp score to [0, 1000] range
  ;; param $score: Input score
  ;; returns: Clamped score
  (func $clamp_score (param $score f64) (result f64)
    ;; Clamp minimum to 0
    (if (f64.lt (local.get $score) (f64.const 0.0))
      (then
        (return (f64.const 0.0))
      )
    )

    ;; Clamp maximum to 1000
    (if (f64.gt (local.get $score) (f64.const 1000.0))
      (then
        (return (f64.const 1000.0))
      )
    )

    (local.get $score)
  )

  ;; Clamp score to [0, 100] range
  ;; param $score: Input score
  ;; returns: Clamped score
  (func $clamp_score_100 (param $score f64) (result f64)
    ;; Clamp minimum to 0
    (if (f64.lt (local.get $score) (f64.const 0.0))
      (then
        (return (f64.const 0.0))
      )
    )

    ;; Clamp maximum to 100
    (if (f64.gt (local.get $score) (f64.const 100.0))
      (then
        (return (f64.const 100.0))
      )
    )

    (local.get $score)
  )

  ;; Debug logging for platform scores
  ;; param $platform_id: Platform identifier
  ;; param $score: Score value
  (func $log_score_debug (param $platform_id i32) (param $score f64)
    ;; This is a no-op in production, but useful for testing
    ;; The OCX runtime can enable/disable logging via compile flags
    (nop)
  )

  ;; ===================================================================
  ;; ADDITIONAL EXPORTS FOR TESTING
  ;; ===================================================================

  ;; Get platform weights (for verification)
  (func $get_weights (result f64 f64 f64)
    (global.get $WEIGHT_GITHUB)
    (global.get $WEIGHT_LINKEDIN)
    (global.get $WEIGHT_UBER)
  )

  ;; Verify weights sum to 1.0 (sanity check)
  (func $verify_weights (result i32)
    (local $sum f64)
    (local.set $sum
      (f64.add
        (global.get $WEIGHT_GITHUB)
        (f64.add
          (global.get $WEIGHT_LINKEDIN)
          (global.get $WEIGHT_UBER))))

    ;; Return 1 if sum equals 1.0, else 0
    (if (result i32) (f64.eq (local.get $sum) (f64.const 1.0))
      (then (i32.const 1))
      (else (i32.const 0))
    )
  )

  ;; ===================================================================
  ;; EXPORTS
  ;; ===================================================================

  (export "compute_reputation" (func $compute_reputation))
  (export "get_weights" (func $get_weights))
  (export "verify_weights" (func $verify_weights))
)
