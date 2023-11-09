import Glossary from 'hew/Glossary';
import React from 'react';

import { JsonObject } from 'types';
import { isObject } from 'utils/data';

type TextTransfomer = (key: string) => string;

interface Props {
  json: JsonObject;
  translateLabel?: TextTransfomer;
}

const JsonGlossary: React.FC<Props> = ({ json, translateLabel }: Props) => {
  const content = Object.entries(json).map(([label, jsonValue]) => {
    let textValue = '';
    if (isObject(jsonValue)) {
      textValue = JSON.stringify(jsonValue, null, 2);
    } else if (jsonValue === '' || jsonValue === null || jsonValue === undefined) {
      textValue = 'N/A';
    } else {
      textValue = jsonValue.toString();
    }
    return {
      label: typeof label === 'string' && translateLabel ? translateLabel(label) : label,
      value: textValue,
    };
  });
  return <Glossary content={content} />;
};

export default JsonGlossary;
