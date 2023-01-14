import {sleep} from 'k6'
import kv from 'k6/x/kv';

export const options = {
  scenarios: {
    generator: {
      exec: 'generator',
      executor: 'per-vu-iterations',
      vus: 5,
    },
    results: {
      exec: 'results',
      executor: 'per-vu-iterations',
      startTime: '1s',
      maxDuration: '2s',
      vus: 1,
    },
    ttl: {
      exec: 'ttl',
      executor: 'constant-vus',
      startTime: '3s',
      vus: 1,
      duration: '5s',
    },
  },
};

const client = new kv.Client('', true);

export function generator() {
  client.set(`hello_${__VU}`, 'world');
  client.setWithTTLInSecond(`ttl_${__VU}`, `ttl_${__VU}`, 5);
}

export function results() {
  try {
    console.log(client.get("hello_1"));
  }
  catch (err) {
    console.log("empty value", err);
  }
  client.delete("hello_1");
  try {
    let keyDeleteValue = client.get("hello_1");
    console.log(typeof (keyDeleteValue));
  }
  catch (err) {
    console.log("empty value", err);
  }
  let r = client.viewPrefix("hello");
  for (var key in r) {
    console.log(key, r[key])
  }
}

export function ttl() {
  let r = client.viewPrefix("ttl");
  let count = 0;
  for (let key in r) {
    count++;
  }
  console.log(`count = ${count}`);
  sleep(1);
}
