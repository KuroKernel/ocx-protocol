import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const responseTime = new Trend('response_time');
const requestCount = new Counter('request_count');

// Stress test configuration
export let options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '2m', target: 50 },   // Stay at 50 users
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '2m', target: 100 },  // Stay at 100 users
    { duration: '1m', target: 200 },  // Ramp up to 200 users
    { duration: '3m', target: 200 },  // Stay at 200 users
    { duration: '1m', target: 500 },  // Ramp up to 500 users
    { duration: '2m', target: 500 },  // Stay at 500 users
    { duration: '1m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% of requests must complete below 1000ms
    http_req_failed: ['rate<0.05'],    // Error rate must be below 5%
    error_rate: ['rate<0.05'],         // Custom error rate below 5%
  },
};

// Test data
const API_BASE_URL = __ENV.API_BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'supersecretkey';

// Sample test artifacts and inputs
const testArtifacts = [
  {
    hash: 'a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456',
    input: '48656c6c6f20576f726c64', // "Hello World" in hex
  },
  {
    hash: 'b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef1234567',
    input: '5465737420496e707574', // "Test Input" in hex
  },
  {
    hash: 'c3d4e5f6789012345678901234567890abcdef1234567890abcdef12345678',
    input: '4c6f61642054657374', // "Load Test" in hex
  },
  {
    hash: 'd4e5f6789012345678901234567890abcdef1234567890abcdef123456789',
    input: '5374726573732054657374', // "Stress Test" in hex
  },
  {
    hash: 'e5f6789012345678901234567890abcdef1234567890abcdef1234567890',
    input: '48696768204c6f6164', // "High Load" in hex
  },
];

// Helper function to generate random test data
function getRandomTestData() {
  const artifact = testArtifacts[Math.floor(Math.random() * testArtifacts.length)];
  return {
    artifact_hash: artifact.hash,
    input: artifact.input,
  };
}

// Helper function to make authenticated request
function makeRequest(method, url, body = null) {
  const headers = {
    'Content-Type': 'application/json',
    'X-API-Key': API_KEY,
  };

  const params = {
    headers: headers,
    timeout: '60s', // Longer timeout for stress test
  };

  let response;
  if (method === 'GET') {
    response = http.get(url, params);
  } else if (method === 'POST') {
    response = http.post(url, JSON.stringify(body), params);
  }

  return response;
}

export default function () {
  group('Health Check', function () {
    const response = makeRequest('GET', `${API_BASE_URL}/health`);
    const success = check(response, {
      'health check status is 200': (r) => r.status === 200,
      'health check response time < 500ms': (r) => r.timings.duration < 500,
    });
    
    errorRate.add(!success);
    responseTime.add(response.timings.duration);
    requestCount.add(1);
  });

  group('Execute Endpoint Stress Test', function () {
    const executeData = getRandomTestData();
    const executeResponse = makeRequest('POST', `${API_BASE_URL}/api/v1/execute`, executeData);
    
    const executeSuccess = check(executeResponse, {
      'execute status is 200, 201, or 429': (r) => r.status === 200 || r.status === 201 || r.status === 429,
      'execute response time < 2000ms': (r) => r.timings.duration < 2000,
    });
    
    errorRate.add(!executeSuccess);
    responseTime.add(executeResponse.timings.duration);
    requestCount.add(1);
  });

  group('Verify Endpoint Stress Test', function () {
    // Create a mock receipt for verification stress test
    const mockReceipt = {
      receipt_data: 'mock_receipt_data_for_stress_test',
      public_key: 'mock_public_key_for_stress_test',
    };
    
    const verifyResponse = makeRequest('POST', `${API_BASE_URL}/verify`, mockReceipt);
    
    const verifySuccess = check(verifyResponse, {
      'verify status is 200 or 400': (r) => r.status === 200 || r.status === 400,
      'verify response time < 1000ms': (r) => r.timings.duration < 1000,
    });
    
    errorRate.add(!verifySuccess);
    responseTime.add(verifyResponse.timings.duration);
    requestCount.add(1);
  });

  group('Receipts Endpoint Stress Test', function () {
    const receiptsResponse = makeRequest('GET', `${API_BASE_URL}/api/v1/receipts/`);
    
    const receiptsSuccess = check(receiptsResponse, {
      'receipts status is 200': (r) => r.status === 200,
      'receipts response time < 1000ms': (r) => r.timings.duration < 1000,
    });
    
    errorRate.add(!receiptsSuccess);
    responseTime.add(receiptsResponse.timings.duration);
    requestCount.add(1);
  });

  // Shorter delay for stress test
  sleep(0.05);
}

export function handleSummary(data) {
  return {
    'stress-test-results.json': JSON.stringify(data, null, 2),
    stdout: `
OCX Protocol Stress Test Results
===============================

Test Duration: ${data.state.testRunDurationMs / 1000}s
Total Requests: ${data.metrics.request_count.values.count}
Error Rate: ${(data.metrics.error_rate.values.rate * 100).toFixed(2)}%
Average Response Time: ${data.metrics.response_time.values.avg.toFixed(2)}ms
95th Percentile: ${data.metrics.response_time.values['p(95)'].toFixed(2)}ms
99th Percentile: ${data.metrics.response_time.values['p(99)'].toFixed(2)}ms
Max Response Time: ${data.metrics.response_time.values.max.toFixed(2)}ms

Thresholds:
- 95th percentile < 1000ms: ${data.metrics.http_req_duration.values['p(95)'] < 1000 ? 'PASS' : 'FAIL'}
- Error rate < 5%: ${data.metrics.http_req_failed.values.rate < 0.05 ? 'PASS' : 'FAIL'}

Stress Test Grade: ${getStressTestGrade(data)}
    `,
  };
}

function getStressTestGrade(data) {
  const errorRate = data.metrics.error_rate.values.rate;
  const p95 = data.metrics.response_time.values['p(95)'];
  
  if (errorRate < 0.01 && p95 < 500) return 'A+ (Excellent)';
  if (errorRate < 0.02 && p95 < 1000) return 'A (Very Good)';
  if (errorRate < 0.05 && p95 < 2000) return 'B (Good)';
  if (errorRate < 0.10 && p95 < 5000) return 'C (Acceptable)';
  return 'D (Needs Improvement)';
}
