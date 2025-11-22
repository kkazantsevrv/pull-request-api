import http from 'k6/http';
import { check } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
  vus: 10,       // 10 виртуальных пользователей (потоков)
  iterations: 100, // Всего 100 запросов
};

export default function () {
  const id = `PR-${Date.now()}-${randomString(5)}`;
  
  const payload = JSON.stringify({
    pull_request_id: id,
    author_id: "5",
    pull_request_name: "Load test k6"
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post('http://localhost:8080/pullRequest/create', payload, params);
  
  check(res, {
    'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
  });
}