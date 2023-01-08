import {sleep} from 'k6'
import {Client} from 'k6/x/kv';

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
      duration: '3s',
    },
  },
};

const client = new Client();

export function generator() {
  client.set(`hello_${__VU}`, 'world');
  client.setWithTTLInSecond(`ttl_${__VU}`, `ttl_${__VU}`, 5);
}

export function results() {
  console.log("Getting hello_1 and then deleting and getting again...");
  console.log(client.get("hello_1"));
  client.delete("hello_1");
  try {
    let keyDeleteValue = client.get("hello_1");
    console.log(typeof (keyDeleteValue));
  }
  catch (err) {
    console.log("empty value", err);
  }
  let r = client.viewPrefix("hello");
  for (let key in r) {
    console.log(key, r[key])
  }
}

export function ttl() {
  for (let k=0; k < 5; k++) {
    try {
      console.log(client.get(`ttl_${k}`));
    } catch (err) {
      console.log(`empty value for 'ttl_${k}'`, err);
    }
  }
  sleep(1);
}