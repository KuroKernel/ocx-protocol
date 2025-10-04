import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const responseTime = new Trend('response_time');
const requestCount = new Counter('request_count');

// Test configuration
export let options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up to 10 users
    { duration: '5m', target: 10 },   // Stay at 10 users
    { duration: '2m', target: 50 },   // Ramp up to 50 users
    { duration: '5m', target: 50 },   // Stay at 50 users
    { duration: '2m', target: 100 },  // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% of requests must complete below 200ms
    http_req_failed: ['rate<0.01'],   // Error rate must be below 1%
    error_rate: ['rate<0.01'],        // Custom error rate below 1%
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
    timeout: '30s',
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
      'health check response time < 100ms': (r) => r.timings.duration < 100,
    });
    
    errorRate.add(!success);
    responseTime.add(response.timings.duration);
    requestCount.add(1);
  });

  group('API Endpoints', function () {
    // Test execute endpoint
    const executeData = getRandomTestData();
    const executeResponse = makeRequest('POST', `${API_BASE_URL}/api/v1/execute`, executeData);
    
    const executeSuccess = check(executeResponse, {
      'execute status is 200 or 201': (r) => r.status === 200 || r.status === 201,
      'execute response time < 500ms': (r) => r.timings.duration < 500,
      'execute response has receipt': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.receipt_cbor !== undefined;
        } catch (e) {
          return false;
        }
      },
    });
    
    errorRate.add(!executeSuccess);
    responseTime.add(executeResponse.timings.duration);
    requestCount.add(1);

    // Test verify endpoint
    if (executeResponse.status === 200 || executeResponse.status === 201) {
      try {
        const executeBody = JSON.parse(executeResponse.body);
        if (executeBody.receipt_cbor) {
          const verifyData = {
            receipt_data: executeBody.receipt_cbor,
            public_key: executeBody.public_key || 'default_public_key',
          };
          
          const verifyResponse = makeRequest('POST', `${API_BASE_URL}/verify`, verifyData);
          
          const verifySuccess = check(verifyResponse, {
            'verify status is 200': (r) => r.status === 200,
            'verify response time < 200ms': (r) => r.timings.duration < 200,
            'verify response is valid': (r) => {
              try {
                const body = JSON.parse(r.body);
                return body.valid !== undefined;
              } catch (e) {
                return false;
              }
            },
          });
          
          errorRate.add(!verifySuccess);
          responseTime.add(verifyResponse.timings.duration);
          requestCount.add(1);
        }
      } catch (e) {
        console.error('Failed to parse execute response:', e);
      }
    }

    // Test artifact info endpoint
    const artifactInfoResponse = makeRequest('GET', `${API_BASE_URL}/api/v1/artifact/info?hash=${executeData.artifact_hash}`);
    
    const artifactInfoSuccess = check(artifactInfoResponse, {
      'artifact info status is 200 or 404': (r) => r.status === 200 || r.status === 404,
      'artifact info response time < 100ms': (r) => r.timings.duration < 100,
    });
    
    errorRate.add(!artifactInfoSuccess);
    responseTime.add(artifactInfoResponse.timings.duration);
    requestCount.add(1);

    // Test receipts endpoint
    const receiptsResponse = makeRequest('GET', `${API_BASE_URL}/api/v1/receipts/`);
    
    const receiptsSuccess = check(receiptsResponse, {
      'receipts status is 200': (r) => r.status === 200,
      'receipts response time < 200ms': (r) => r.timings.duration < 200,
    });
    
    errorRate.add(!receiptsSuccess);
    responseTime.add(receiptsResponse.timings.duration);
    requestCount.add(1);
  });

  group('Swagger Documentation', function () {
    const swaggerResponse = makeRequest('GET', `${API_BASE_URL}/swagger/`);
    
    const swaggerSuccess = check(swaggerResponse, {
      'swagger status is 200': (r) => r.status === 200,
      'swagger response time < 500ms': (r) => r.timings.duration < 500,
    });
    
    errorRate.add(!swaggerSuccess);
    responseTime.add(swaggerResponse.timings.duration);
    requestCount.add(1);
  });

  // Small delay between requests
  sleep(0.1);
}

export function handleSummary(data) {
  return {
    'loadtest-results.json': JSON.stringify(data, null, 2),
    stdout: `
OCX Protocol Load Test Results
==============================

Test Duration: ${data.state.testRunDurationMs / 1000}s
Total Requests: ${data.metrics.request_count.values.count}
Error Rate: ${(data.metrics.error_rate.values.rate * 100).toFixed(2)}%
Average Response Time: ${data.metrics.response_time.values.avg.toFixed(2)}ms
95th Percentile: ${data.metrics.response_time.values['p(95)'].toFixed(2)}ms
99th Percentile: ${data.metrics.response_time.values['p(99)'].toFixed(2)}ms

Thresholds:
- 95th percentile < 200ms: ${data.metrics.http_req_duration.values['p(95)'] < 200 ? 'PASS' : 'FAIL'}
- Error rate < 1%: ${data.metrics.http_req_failed.values.rate < 0.01 ? 'PASS' : 'FAIL'}

Performance Grade: ${getPerformanceGrade(data)}
    `,
  };
}

function getPerformanceGrade(data) {
  const errorRate = data.metrics.error_rate.values.rate;
  const p95 = data.metrics.response_time.values['p(95)'];
  
  if (errorRate < 0.001 && p95 < 100) return 'A+ (Excellent)';
  if (errorRate < 0.005 && p95 < 200) return 'A (Very Good)';
  if (errorRate < 0.01 && p95 < 500) return 'B (Good)';
  if (errorRate < 0.05 && p95 < 1000) return 'C (Acceptable)';
  return 'D (Needs Improvement)';
}
