import React, { PropsWithChildren } from 'react';

import { toHtmlId } from 'utils/string';

import css from './Section.module.scss';

interface Props {
  bodyBorder?: boolean;
  divider?: boolean;
  hideTitle?: boolean;
  maxHeight?: boolean;
  options?: React.ReactNode;
  title: string;
}

const defaultProps = { divider: false };

const Section: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const bodyClasses = [ css.body ];

  if (props.divider) classes.push(css.divider);
  if (props.maxHeight) classes.push(css.maxHeight);
  if (props.bodyBorder) bodyClasses.push(css.bodyBorder);

  return (
    <section className={classes.join(' ')} id={toHtmlId(props.title)}>
      <div className={css.header}>
        {!props.hideTitle && <h5 className={css.title}>{props.title}</h5>}
        {props.options && <div className={css.options}>{props.options}</div>}
      </div>
      <div className={bodyClasses.join(' ')}>
        {props.children}
      </div>
    </section>
  );
};

Section.defaultProps = defaultProps;

export default Section;
