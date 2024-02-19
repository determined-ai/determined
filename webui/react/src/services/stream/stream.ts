import dayjs, { Dayjs } from 'dayjs';
import { forEach, map, reduce, trimEnd } from 'lodash';

import rootLogger from 'utils/Logger';

import { decode_keys, KeyCache } from './keyCache';

import { Streamable, StreamContent, StreamEntityMap, StreamSpec } from '.';

const logger = rootLogger.extend('services', 'stream');

// About 60 seconds of auto-retry.
const backoffs = [0, 1, 2, 4, 8, 10, 10, 10, 15];

type Subscription = {
  keyCache: KeyCache;
  spec: StreamSpec;
  id?: string;
};

type SubscriptionGroup = Partial<Record<Streamable, Subscription>>;

export class Stream {
  readonly #wsUrl: string;
  #ws?: WebSocket = undefined;
  #retries: number = 0;
  #timeout: Dayjs = dayjs();
  #numSyncs: number = 0;
  #closedByClient: boolean = false;

  #subs: Array<SubscriptionGroup> = [];
  #curSub?: SubscriptionGroup;

  // syncSent updates when msg sent to the stream
  #syncSent?: string = undefined;
  // syncStarted updates when recieving msg of {sync_id, complated: false}
  #syncStarted?: string = undefined;
  // syncStarted updates when recieving msg of {sync_id, complated: true}
  #syncComplete?: string = undefined;
  // List of messages recieved from server
  #pendingMsg: Array<Record<string, StreamContent>> = [];

  //callbacks
  #onUpsert: (m: Record<string, StreamContent>) => void;
  #onDelete: (s: Streamable, a: Array<number>) => void;
  #isLoaded?: (ids: Array<string>) => void;

  constructor(
    wsUrl: string,
    onUpsert: (m: Record<string, StreamContent>) => void,
    onDelete: (s: Streamable, a: Array<number>) => void,
    isLoaded?: (ids: Array<string>) => void,
  ) {
    this.#wsUrl = wsUrl;
    this.#onUpsert = onUpsert;
    this.#onDelete = onDelete;
    this.#isLoaded = isLoaded;
    this.#advance();
  }

  #connect(): WebSocket {
    const ws = new WebSocket(this.#wsUrl);
    ws.onopen = () => {
      logger.info('Streaming websocket opened!');
      this.#retries = 0;
      this.#advance();
    };
    ws.onerror = (err) => {
      // No need to do anything else becauses onerror will trigger onclose
      logger.error('Streaming websocket errored: ', err);
    };

    ws.onclose = () => {
      this.#syncSent = undefined;
      const backoff = backoffs[this.#retries];
      if (backoff === undefined) {
        throw new Error('Websocket cannot reconnect!');
      }
      this.#timeout = dayjs().add(backoff, 'second');
      this.#retries += 1;
      logger.info(`#${this.#retries} of retries: in ${backoff}s`);
      setTimeout(() => this.#advance(), backoff * 1000);
    };

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data) as Record<string, StreamContent>;
      this.#pendingMsg.push(msg);
      this.#advance();
    };

    return ws;
  }

  #shouldSkip(newSub: SubscriptionGroup): boolean {
    if (!this.#curSub) return false;
    let same = true;
    forEach(newSub, (val, k) => {
      if (!this.#curSub?.[k as Streamable]?.spec.equals(val?.spec)) same = false;
    });
    return same;
  }

  #sendSpec(newSub: SubscriptionGroup): void {
    // Skip current sub and move to next only if sub has already been sent
    if (this.#shouldSkip(newSub) && this.#syncSent) {
      this.#advance();
      return;
    }

    this.#numSyncs += 1;
    const sync_id = this.#numSyncs.toString();
    const spec = reduce(
      newSub,
      (spec, ent, k) => {
        if (!ent) return spec;
        spec['known'][k] = ent.keyCache.known();
        spec['subscribe'][k] = {
          ...ent.spec.toWire(),
          since: ent.keyCache.maxSeq(),
        };
        return spec;
      },
      { known: {}, subscribe: {}, sync_id: sync_id } as Record<string, StreamContent>,
    );

    this.#curSub = newSub;
    this.#ws!.send(JSON.stringify(spec));
    this.#syncSent = sync_id;
  }

  #processPending(): void {
    while (this.#pendingMsg.length > 0) {
      const msg = this.#pendingMsg.shift();
      if (!msg) break;
      if (msg['sync_id']) {
        if (!msg['complete']) {
          this.#syncStarted = msg['sync_id'];
        } else {
          const completedSpecId = map(this.#curSub, (spec) => spec?.id || '').filter((i) => !!i);
          this.#isLoaded?.(completedSpecId);
          this.#syncComplete = msg['sync_id'];
          this.#advance();
        }
      } else if (this.#syncSent !== this.#syncStarted) {
        // Ignore all messages between when we send a new subscription and when the
        // sync-start message for that subscription arrives.  These are the online
        // updates for a subscription we no longer care about.
      } else {
        forEach(msg, (val, k) => {
          if (k.includes('_deleted')) {
            const stream_key = trimEnd(k, '_deleted') as Streamable;
            const deleted_keys = decode_keys(val as string);

            this.#curSub?.[stream_key]?.keyCache.delete_msg(deleted_keys);
            this.#onDelete(stream_key, deleted_keys);
          } else {
            this.#curSub?.[StreamEntityMap[k]]?.keyCache.upsert([val.id], val.seq);
            this.#onUpsert(msg);
          }
        });
      }
    }
  }

  #processSubscription() {
    if (!this.#curSub && this.#subs.length === 0) return;

    let spec: SubscriptionGroup | undefined;

    if (!this.#syncSent) {
      // The websocket just connected/reconnected
      // Resend current subscription if current sync not completed
      if (this.#curSub && (!this.#syncComplete || this.#syncStarted !== this.#syncComplete)) {
        spec = this.#curSub;
      } else {
        spec = this.#subs.shift();
      }
      spec && this.#sendSpec(spec);
      return;
    }

    if (this.#subs.length > 0 && this.#syncComplete === this.#syncSent) {
      // For established connection, only send a new sub when current sub is completed
      spec = this.#subs.shift();
      this.#sendSpec(spec!);
    }
  }

  #advance(): void {
    if (this.#closedByClient) {
      // We want to shut down
      if (this.#ws && this.#ws.readyState !== this.#ws.CLOSED) {
        if (this.#ws.readyState !== this.#ws.CLOSING) {
          this.#ws.close();
        }
      }
      return;
    }

    if (this.#ws && this.#ws.readyState === this.#ws.CLOSED) {
      // Our websocket broke and we wait till timeout finish to reconnect
      if (dayjs().isBefore(this.#timeout)) return;
      this.#ws = undefined;
    }

    if (!this.#ws) {
      this.#ws = this.#connect();
    }

    if (this.#ws.readyState !== this.#ws.OPEN) {
      return;
    }
    this.#processPending();
    this.#processSubscription();
  }

  public subscribe(spec: StreamSpec, id?: string): void {
    const curSpec = this.#curSub?.[spec.id()];
    if (curSpec && curSpec.spec.equals(spec)) return;
    const keyCache = new KeyCache();
    const newSpec = { [spec.id()]: { id, keyCache, spec } };
    this.#curSub ? this.#subs.push({ ...this.#curSub, ...newSpec }) : this.#subs.push(newSpec);
    this.#advance();
  }

  public close(): void {
    this.#closedByClient = true;
    this.#advance();
  }
}
