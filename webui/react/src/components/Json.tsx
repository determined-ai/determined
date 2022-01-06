import React from 'react';

import { RawJson } from 'types';
import { isObject } from 'utils/data';

import css from './Json.module.scss';

type TextTransfomer = (key: string) => string;

interface Props {
  json: RawJson;
  translateLabel?: TextTransfomer;
}

interface RowProps {
  tag: string;
  translateLabel?: TextTransfomer;
  value: RawJson | string | number | null;
}

const Row: React.FC<RowProps> = ({ translateLabel, tag, value }: RowProps) => {
  let textValue = '';
  if (isObject(value)) {
    textValue = JSON.stringify(value, null, 2);
  } else if (value === '' || value === null || value === undefined) {
    textValue = 'N/A';
  } else {
    textValue = value.toString();
  }
  return (
    <li className={css.item} key={tag}>
      <span className={css.tag}>
        {typeof tag === 'string' && translateLabel ? translateLabel(tag) : tag}
        :
      </span>
      <span className={css.value}>
        {textValue}
      </span>
    </li>
  );
};

const Json: React.FC<Props> = ({ json, translateLabel }: Props) => {

  return (
    <ul className={css.base}>
      {Object.entries(json).map(([ tag, value ]) => (
        <Row key={tag} tag={tag} translateLabel={translateLabel} value={value} />
      ))}
    </ul>
  );
};

export default Json;
