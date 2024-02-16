import dayjs, { Dayjs } from 'dayjs';
import { forEach, reduce, trimEnd } from 'lodash';

import rootLogger from 'utils/Logger';

import { decode_keys, KeyCache } from './keyCache';
import { StreamSpec } from './projects';

import { Streamable, StreamContent, StreamEntityMap } from '.';

const logger = rootLogger.extend('services', 'stream');

// About 60 seconds of auto-retry.
const backoffs = [0, 1, 2, 4, 8, 10, 10, 10, 15];
export const sleep = (s: number): Promise<unknown> => new Promise((r) => setTimeout(r, 1000 * s));

type Subscription = {
  keyCache: KeyCache;
  spec: StreamSpec;
};

export class Stream {
  readonly #wsUrl: string;
  #ws: WebSocket | undefined = undefined;
  #retries: number = 0;
  #timeout: Dayjs = dayjs();
  #numSyncs: number = 0;
  #closedByClient: boolean = false;

  #subs: Array<Record<Streamable, Subscription>> = [];
  #curSub: Record<Streamable, Subscription> | undefined;

  // syncSent updates when msg sent to the stream
  #syncSent: string | undefined = undefined;
  // syncStarted updates when recieving msg of {sync_id, complated: false}
  #syncStarted: string | undefined = undefined;
  // syncStarted updates when recieving msg of {sync_id, complated: true}
  #syncComplete: string | undefined = undefined;
  // List of messages recieved from server
  #pendingMsg: Array<Record<string, StreamContent>> = [];

  //callbacks
  #onUpsert: (m: Record<string, StreamContent>) => void;
  #onDelete: (s: Streamable, a: Array<number>) => void;
  #isLoading: ((b: boolean) => void) | undefined;

  constructor(
    wsUrl: string,
    onUpsert: (m: Record<string, StreamContent>) => void,
    onDelete: (s: Streamable, a: Array<number>) => void,
    isLoading?: (b: boolean) => void,
  ) {
    this.#wsUrl = wsUrl;
    this.#onUpsert = onUpsert;
    this.#onDelete = onDelete;
    this.#isLoading = isLoading;
    this.#advance();
  }

  #connect() {
    this.#ws = new WebSocket(this.#wsUrl);
    this.#ws.onopen = () => {
      logger.info('Streaming websocket opened!');
      this.#retries = 0;
      this.#advance();
    };
    this.#ws.onerror = (err) => {
      // No need to do anything else becauses onerror will trigger onclose
      logger.error('Streaming websocket errored: ', err);
    };

    this.#ws.onclose = () => {
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

    this.#ws.onmessage = (event) => {
      const msg = JSON.parse(event.data) as Record<string, StreamContent>;
      this.#pendingMsg.push(msg);
      this.#advance();
    };
  }

  #shouldSkip(newSub: Record<Streamable, Subscription>): boolean {
    if (!this.#curSub) return false;
    let same = true;
    forEach(newSub, (val, k) => {
      if (!this.#curSub?.[k as Streamable].spec.equals(val.spec)) same = false;
    });
    return same;
  }

  #sendSpec(newSub?: Record<Streamable, Subscription>): void {
    if (!newSub) return;
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
    this.#ws && this.#ws.send(JSON.stringify(spec));
    this.#syncSent = sync_id;
  }

  #processMsg(): void {
    while (this.#pendingMsg.length > 0) {
      const msg = this.#pendingMsg.shift();
      if (!msg) break;
      if (msg['sync_id']) {
        if (!msg['complete']) {
          this.#syncStarted = msg['sync_id'];
          this.#isLoading?.(false);
        } else {
          this.#syncComplete = msg['sync_id'];
          this.#isLoading?.(true);
        }
      } else if (this.#syncSent === this.#syncStarted) {
        // Ignore all messages between when we send a new subscription and when the
        // sync-start message for that subscription arrives.  These are the online
        // updates for a subscription we no longer care about.

        forEach(msg, (val, k) => {
          if (k.includes('_deleted')) {
            const stream_key = trimEnd(k, '_deleted') as Streamable;
            const deleted_keys = decode_keys(val as string);

            this.#curSub?.[stream_key].keyCache.delete_msg(deleted_keys);
            this.#onDelete(stream_key, deleted_keys);
          } else {
            this.#curSub?.[StreamEntityMap[k]].keyCache.upsert([val.id], val.seq);
            this.#onUpsert(msg);
          }
        });
      }
    }
  }

  #processSubscription() {
    if (!this.#curSub && (!this.#subs || this.#subs.length === 0)) return;

    let spec: Record<Streamable, Subscription> | undefined;
    // Just connected/reconnected
    if (!this.#syncSent) {
      // Resend current subscription if current sync not completed
      if (this.#curSub && (!this.#syncComplete || this.#syncStarted !== this.#syncComplete)) {
        spec = this.#curSub;
      } else {
        spec = this.#subs.shift();
      }
      this.#sendSpec(spec);
    } else if (this.#subs && this.#syncComplete === this.#syncSent) {
      // for established connection, only send a new sub when current sub is completed
      spec = this.#subs.shift();
      this.#sendSpec(spec);
    }
  }

  #advance(): void {
    // We want to shut down
    if (this.#closedByClient) {
      if (this.#ws && this.#ws.readyState !== this.#ws.CLOSED) {
        if (this.#ws.readyState !== this.#ws.CLOSING) {
          this.#ws.close();
        }
      }
      return;
    }

    // We got shut down and we wait till timeout finish to reconnect
    if (this.#ws && this.#ws.readyState === this.#ws.CLOSED) {
      if (dayjs().isBefore(this.#timeout)) return;
      this.#ws = undefined;
    }

    if (!this.#ws) {
      this.#connect();
    } else if (this.#ws.readyState === this.#ws.OPEN) {
      this.#processMsg();
      this.#processSubscription();
    }
  }

  public subscribe(spec: StreamSpec): void {
    const curSpec = this.#curSub?.[spec.id()];
    const keyCache = curSpec && !curSpec.spec.equals(spec) ? curSpec.keyCache : new KeyCache();

    this.#subs.push({ ...this.#curSub, [spec.id()]: { keyCache, spec } });

    this.#advance();
  }

  public close(): void {
    this.#closedByClient = true;
    this.#advance();
  }
}
