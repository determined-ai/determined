import React from 'react';

import { CommonProps } from 'types';
import { toHtmlId } from 'utils/string';

import css from './Page.module.scss';

interface Props extends CommonProps {
  title: string;
  hideTitle?: boolean;
  maxHeight?: boolean;
}

const defaultProps = {
  hideTitle: false,
};

const Page: React.FC<Props> = (props: Props) => {
  const classes = [ props.className, css.base ];

  if (props.maxHeight) classes.push(css.maxHeight);

  return (
    <main className={classes.join(' ')} id={toHtmlId(props.title)}>
      {props.hideTitle || <h5 className={css.title}>{props.title}</h5>}
      <div className={css.body}>
        {props.children}
      </div>
    </main>
  );
};

Page.defaultProps = defaultProps;

export default Page;
