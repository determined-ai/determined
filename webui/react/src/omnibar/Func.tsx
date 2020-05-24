import React from 'react';

interface FuncEntry {
  name: string;
  fn: QueryableFunc;
}

type QueryableFunc = (a: string) =>  void|Promise<void>;

interface ResultItem {
  title: string; // to utilize the default renderer
  onAction: Action<ResultItem>;
}

const funcs: FuncEntry[]  = [
  {
    fn: alert,
    name: 'alert',
  },
  {
    fn: console.log,
    name: 'log',
  },
];

type Action<T> = (arg?: T) => void

export const funcExt = (query: string): ResultItem[] => {
  const sections = query.split(' ');
  if (sections.length > 2) return []; // we dont support this for now
  const matchingFuncs = funcs.filter(it => it.name.match(new RegExp(sections[0], 'i')));
  return matchingFuncs.map(it => ({
    title: `${it.name}(${sections[1]})`,
    onAction: () => {
      console.log('calling this func', it.name);
      it.fn.call(null, sections[1]);
      // it.fn(sections[1]);
    },
  }));
};

export const funcOnAction = (it: any): void => it.onAction.bind(it)(it);
