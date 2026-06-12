import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const loginDuration = new Trend('login_duration', true);

const BASE_URL = 'http://localhost:8080';
const TEST_USER = { email: 'perf-auth@test.com', password: 'PerfTest123!' };

export const options = {
  stages: [
    { duration: '15s', target: 20 },
    { duration: '30s', target: 20 },
    { duration: '15s', target: 50 },
    { duration: '30s', target: 50 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed:   ['rate<0.05'],
    http_req_duration: ['p(95)<1000'],
    login_duration:    ['p(95)<800'],
  },
};

export function setup() {
  const res = http.post(
    `${BASE_URL}/gateway/auth/register`,
    JSON.stringify(TEST_USER),
    { headers: { 'Content-Type': 'application/json' } }
  );

  console.log(`Register status: ${res.status} — ${res.body}`);
}

export default function () {
  const loginStart = Date.now();
  const loginRes = http.post(
    `${BASE_URL}/gateway/auth/login`,
    JSON.stringify(TEST_USER),
    { headers: { 'Content-Type': 'application/json' } }
  );

  loginDuration.add(Date.now() - loginStart);

  const loginOk = check(loginRes, {
    'login status 200':        (r) => r.status === 200,
    'access_token header set': (r) => r.headers['Access_token'] !== undefined && r.headers['Access_token'] !== '',
  });

  errorRate.add(!loginOk);

  sleep(1);
}