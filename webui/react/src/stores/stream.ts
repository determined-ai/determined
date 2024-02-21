import { forEach } from 'lodash';

import { serverAddress } from 'routes/utils';
import { Streamable, StreamContent, StreamEntityMap, StreamSpec } from 'services/stream';
import { Stream } from 'services/stream/stream';

export interface StreamSubscriber {
  upsert: (content: StreamContent) => void;
  delete: (id: number) => void;
  isLoaded?: (ids: Array<string>) => void;
  id: () => Streamable;
}

class StreamStore {
  #stream: Stream;
  #subscribers: Partial<Record<Streamable, StreamSubscriber>> = {};

  constructor() {
    const socketUrl = `${serverAddress().replace('http', 'ws')}/stream`;
    const onUpsert = (m: Record<string, StreamContent>) => {
      forEach(m, (val, k) => {
        this.#subscribers[StreamEntityMap[k]]?.upsert(val);
      });
    };

    const onDelete = (entity: Streamable, deleted: Array<number>) => {
      if (deleted.length === 0) return;
      for (const d of deleted) {
        this.#subscribers[entity]?.delete(d);
      }
    };

    const isLoaded = (ids: Array<string>) => {
      forEach(this.#subscribers, (sub) => {
        sub?.isLoaded?.(ids);
      });
    };
    this.#stream = new Stream(socketUrl, onUpsert, onDelete, isLoaded);
  }

  public on(sub: StreamSubscriber) {
    this.#subscribers[sub.id()] = sub;
  }

  public off(key: Streamable) {
    delete this.#subscribers[key];
  }

  public emit(spec: StreamSpec, id?: string) {
    this.#stream.subscribe(spec, id);
  }
}

export default new StreamStore();
