import React from 'react';

import css from './Section.module.scss';

interface Props {
  children: React.ReactNode;
  divider?: boolean;
  title?: string | React.ReactNode;
}
const LETTERS = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
const CHARACTERS = `0123456789${LETTERS}`;
const DEFAULT_ALPHA_NUMERIC_LENGTH = 8;
const isString = (data: unknown): data is string => typeof data === 'string';
const generateAlphaNumeric = (
  length = DEFAULT_ALPHA_NUMERIC_LENGTH,
  chars = CHARACTERS,
): string => {
  let result = '';
  for (let i = length; i > 0; --i) {
    result += chars[Math.floor(Math.random() * chars.length)];
  }
  return result;
};
const toHtmlId = (str: string): string => {
  return str
    .replace(/[\s_]/gi, '-')
    .replace(/[^a-z0-9-]/gi, '')
    .toLowerCase();
};

const defaultProps = { divider: false };

const Section: React.FC<Props> = (props: Props) => {
  const id = isString(props.title) ? toHtmlId(props.title) : generateAlphaNumeric();
  const classes = [css.base];
  const titleClasses = [css.title];

  if (props.divider) classes.push(css.divider);
  if (typeof props.title === 'string') titleClasses.push(css.string);

  return (
    <section className={classes.join(' ')} id={id}>
      {!!props.title && (
        <div className={css.header}>
          <h5 className={titleClasses.join(' ')}>{props.title}</h5>
        </div>
      )}
      <div className={css.body}>{props.children}</div>
    </section>
  );
};

Section.defaultProps = defaultProps;

export default Section;
