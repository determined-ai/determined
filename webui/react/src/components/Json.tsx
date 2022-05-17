import React from 'react';

import { isObject } from 'shared/utils/data';

import { RawJson } from '../shared/types';

import css from './Json.module.scss';

type TextTransfomer = (key: string) => string;

interface Props {
  alternateBackground?: boolean;
  hideDivider?: boolean;
  json: RawJson;
  translateLabel?: TextTransfomer;
}

interface RowProps {
  label: string;
  translateLabel?: TextTransfomer;
  value: RawJson | string | number | null;
}

const Row: React.FC<RowProps> = ({ translateLabel, label, value }: RowProps) => {
  let textValue = '';
  if (isObject(value)) {
    textValue = JSON.stringify(value, null, 2);
  } else if (value === '' || value === null || value === undefined) {
    textValue = 'N/A';
  } else {
    textValue = value.toString();
  }
  return (
    <li className={css.item} key={label}>
      <span className={css.label}>
        {typeof label === 'string' && translateLabel ? translateLabel(label) : label}
        :
      </span>
      <span className={css.value}>
        {textValue}
      </span>
    </li>
  );
};

const Json: React.FC<Props> = (
  { json, translateLabel, hideDivider, alternateBackground }: Props,
) => {
  const classes = [ css.base ];
  if(hideDivider) classes.push(css.hideDivider);
  if(alternateBackground) classes.push(css.alternateBackground);
  return (
    <ul className={classes.join(' ')}>
      {Object.entries(json).map(([ label, value ]) => (
        <Row key={label} label={label} translateLabel={translateLabel} value={value} />
      ))}
    </ul>
  );
};

export default Json;
