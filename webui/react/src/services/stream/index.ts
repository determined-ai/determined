export type Streamable = 'projects' | 'experiments';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export type StreamContent = any;

export const StreamEntityMap: Record<string, Streamable> = {
  experiment: 'experiments',
  project: 'projects',
};

export abstract class StreamSpec {
  abstract copy: () => StreamSpec;
  abstract equals: (sp?: StreamSpec) => boolean;
  abstract id: () => Streamable;
  abstract toWire: () => Record<string, Array<number>>;
}
