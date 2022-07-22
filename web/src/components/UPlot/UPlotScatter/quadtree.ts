export const pointWithin = (
  px: number,
  py: number,
  rlft: number,
  rtop: number,
  rrgt: number,
  rbtm: number,
): boolean => {
  return px >= rlft && px <= rrgt && py >= rtop && py <= rbtm;
};

type QuadTreeCallback = (q: QuadTree) => void;

const MAX_OBJECTS = 10;
const MAX_LEVELS = 4;

class QuadTree {
  x: number;
  y: number;
  w: number;
  h: number;
  l: number;
  o: QuadTree[];
  q: QuadTree[] | null;
  seriesIndex?: number;
  dataIndex?: number;

  constructor(
    x: number,
    y: number,
    w: number,
    h: number,
    l?: number,
    seriesIndex?: number,
    dataIndex?: number,
  ) {
    this.x = x;
    this.y = y;
    this.w = w;
    this.h = h;
    this.l = l || 0;
    this.o = [];
    this.q = null;
    this.seriesIndex = seriesIndex;
    this.dataIndex = dataIndex;
  }

  add(o: QuadTree): void {
    if (this.q != null) {
      this.quads(o.x, o.y, o.w, o.h, (q) => q.add(o));
    } else {
      const os = this.o;

      os.push(o);

      if (os.length > MAX_OBJECTS && this.l < MAX_LEVELS) {
        this.split();

        for (let i = 0; i < os.length; i++) {
          const oi = os[i];
          this.quads(oi.x, oi.y, oi.w, oi.h, (q) => q.add(oi));
        }

        this.o.length = 0;
      }
    }
  }

  clear(): void {
    this.o.length = 0;
    this.q = null;
  }

  get(x: number, y: number, w: number, h: number, cb: QuadTreeCallback): void {
    const os = this.o;

    for (let i = 0; i < os.length; i++) cb(os[i]);

    if (this.q !== null) {
      this.quads(x, y, w, h, (q) => q.get(x, y, w, h, cb));
    }
  }

  // invokes callback with index of each overlapping quad
  quads(x: number, y: number, w: number, h: number, cb: QuadTreeCallback): void {
    if (this.q === null) return;

    const q = this.q;
    const hzMid = this.x + this.w / 2;
    const vtMid = this.y + this.h / 2;
    const startIsNorth = y < vtMid;
    const startIsWest = x < hzMid;
    const endIsEast = x + w > hzMid;
    const endIsSouth = y + h > vtMid;

    // top-right quad
    startIsNorth && endIsEast && cb(q[0]);
    // top-left quad
    startIsWest && startIsNorth && cb(q[1]);
    // bottom-left quad
    startIsWest && endIsSouth && cb(q[2]);
    // bottom-right quad
    endIsEast && endIsSouth && cb(q[3]);
  }

  split(): void {
    const x = this.x;
    const y = this.y;
    const w = this.w / 2;
    const h = this.h / 2;
    const l = this.l + 1;

    this.q = [
      // top right
      new QuadTree(x + w, y, w, h, l),
      // top left
      new QuadTree(x, y, w, h, l),
      // bottom left
      new QuadTree(x, y + h, w, h, l),
      // bottom right
      new QuadTree(x + w, y + h, w, h, l),
    ];
  }
}

export default QuadTree;
