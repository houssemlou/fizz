import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const errorRate = new Rate("errors");
const fizzbuzzTrend = new Trend("fizzbuzz_duration");
const statsTrend = new Trend("stats_duration");

export const options = {
  stages: [
    { duration: "15s", target: 20 },  // ramp up
    { duration: "30s", target: 20 },  // hold
    { duration: "10s", target: 0 },   // ramp down
  ],
  thresholds: {
    http_req_failed:      ["rate<0.01"],       // < 1% errors
    http_req_duration:    ["p(95)<300"],        // 95% of requests under 300ms
    fizzbuzz_duration:    ["p(95)<300"],
    stats_duration:       ["p(95)<100"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8081";
const API_KEY  = __ENV.API_KEY  || "change-me";

const headers = {
  "X-API-Key": API_KEY,
};

// A handful of different param sets to generate variety in the stats table.
const paramSets = [
  { int1: 3, int2: 5, limit: 100, str1: "fizz",  str2: "buzz" },
  { int1: 2, int2: 7, limit: 50,  str1: "foo",   str2: "bar"  },
  { int1: 4, int2: 6, limit: 200, str1: "ping",  str2: "pong" },
  { int1: 3, int2: 9, limit: 75,  str1: "hello", str2: "world" },
];

export default function () {
  const params = paramSets[Math.floor(Math.random() * paramSets.length)];
  const qs = `int1=${params.int1}&int2=${params.int2}&limit=${params.limit}&str1=${params.str1}&str2=${params.str2}`;

  // --- /v1/fizzbuzz ---
  const fizzbuzzRes = http.get(`${BASE_URL}/v1/fizzbuzz?${qs}`, { headers });
  fizzbuzzTrend.add(fizzbuzzRes.timings.duration);
  errorRate.add(fizzbuzzRes.status !== 200);
  check(fizzbuzzRes, {
    "fizzbuzz: status 200":           (r) => r.status === 200,
    "fizzbuzz: result is array":      (r) => Array.isArray(r.json("result")),
    "fizzbuzz: result non-empty":     (r) => r.json("result").length > 0,
  });

  // --- /v1/stats ---
  const statsRes = http.get(`${BASE_URL}/v1/stats`, { headers });
  statsTrend.add(statsRes.timings.duration);
  errorRate.add(statsRes.status !== 200);
  check(statsRes, {
    "stats: status 200": (r) => r.status === 200,
  });

  // --- /v1/health (no auth needed) ---
  const healthRes = http.get(`${BASE_URL}/v1/health`);
  check(healthRes, {
    "health: status 200":   (r) => r.status === 200,
    "health: status is ok": (r) => r.json("status") === "ok",
  });

  sleep(0.5);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data),
  };
}

function textSummary(data) {
  const m = data.metrics;
  const p95 = (metric) =>
    metric ? metric.values["p(95)"].toFixed(2) + "ms" : "n/a";
  const rate = (metric) =>
    metric ? (metric.values.rate * 100).toFixed(2) + "%" : "n/a";

  return `
=== FizzBuzz Load Test Summary ===
  Total requests : ${m.http_reqs?.values.count ?? 0}
  Error rate     : ${rate(m.errors)}
  p95 overall    : ${p95(m.http_req_duration)}
  p95 /fizzbuzz  : ${p95(m.fizzbuzz_duration)}
  p95 /stats     : ${p95(m.stats_duration)}
`;
}
