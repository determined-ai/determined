export function decode_keys(keys: string): Array<number> {
  const retval: Array<number> = [];
  if (keys.length === 0) {
    return retval;
  }
  for (const key of keys.split(',')) {
    if (key.includes('-')) {
      const [start, end] = key.split('-');
      for (let i = Number(start); i <= Number(end); i++) {
        retval.push(i);
      }
    } else {
      retval.push(Number(key));
    }
  }
  return retval;
}

export function encode_keys(set: Set<number>): string {
  if (!set || set.size === 0) return '';
  const out: Array<string> = [];
  const keys = Array.from(set)
    .sort((l, r) => l - r)
    .values();
  let start: number = keys.next().value;
  let end = start;

  const emit = (start: number, end: number, out: Array<string>) => {
    if (start === end) {
      out.push(start.toString());
    } else {
      out.push(`${start}-${end}`);
    }
  };

  for (const k of keys) {
    if (k === end + 1) {
      end = k;
      continue;
    }
    // end of a range
    emit(start, end, out);
    start = k;
    end = start;
  }
  emit(start, end, out);

  return out.join(',');
}

export class KeyCache {
  #keys: Set<number>;
  #maxseq: number;

  constructor(keys?: Set<number>) {
    this.#keys = keys || new Set();
    this.#maxseq = 0;
  }

  #delete_one(id: number) {
    this.#keys.delete(id);
  }

  public upsert(ids: Array<number>, seq: number): void {
    ids.forEach((id) => this.#keys.add(id));
    this.#maxseq = Math.max(this.#maxseq, seq);
  }

  public delete_msg(deleted: Array<number>): void {
    for (const id of deleted) {
      this.#delete_one(id);
    }
  }

  public known(): string {
    return encode_keys(this.#keys);
  }

  public maxSeq(): number {
    return this.#maxseq;
  }
}
