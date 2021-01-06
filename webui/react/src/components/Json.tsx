import React from 'react';

import { RawJson } from 'types';
import { isObject } from 'utils/data';

import css from './Json.module.scss';

type TextTransfomer = (key: string) => string;

interface Props {
  json: RawJson;
  transformLabel?: TextTransfomer;
}

const row = (
  label: string,
  value: RawJson | string | number | null,
  transformLabel?: TextTransfomer,
): React.ReactNode => {
  let textValue = '';
  if (isObject(value)) {
    textValue = JSON.stringify(value, null, 2);
  } else if (value === '' || value === null || value === undefined) {
    textValue = 'N/A';
  } else {
    textValue = value.toString();
  }
  return <li className={css.item} key={label}>
    <span className={css.label}>
      {typeof label === 'string' && transformLabel ? transformLabel(label) : label}
      :</span>
    <span className={css.value}>{textValue}</span>
  </li>;
};

const Json: React.FC<Props> = ({ json, transformLabel }: Props) => {

  return (
    <ul className={css.base}>
      {Object.entries(json).map(([ label, value ]) => row(label, value, transformLabel))}
    </ul>
  );
};

export default Json;
