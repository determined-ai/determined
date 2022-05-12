import React, { MouseEvent, PropsWithChildren, useCallback } from 'react';

import { handlePath, linkPath } from 'routes/utils';

import { AnyMouseEventHandler, windowOpenFeatures } from '../shared/utils/routes';

import css from './Link.module.scss';

export interface Props {
  className?: string;
  disabled?: boolean;
  // is this external to the assets hosted by React?
  external?: boolean;
  inherit?: boolean;
  isButton?: boolean;
  label?: string;
  onClick?: AnyMouseEventHandler;
  path?: string;
  popout?: boolean;
  rawLink?: boolean;
  size?: 'tiny' | 'small' | 'medium' | 'large';
}

const Link: React.FC<Props> = ({
  external,
  popout,
  onClick,
  ...props
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const rel = windowOpenFeatures.join(' ');

  if (props.className) classes.push(props.className);
  if (props.disabled) classes.push(css.disabled);
  if (props.inherit) classes.push(css.inherit);
  if (props.isButton) classes.push('ant-btn');
  if (props.size) classes.push(css[props.size]);

  const href = props.path ? linkPath(props.path, external) : undefined;
  const handleClick = useCallback((event: MouseEvent) => {
    handlePath(event, { external, onClick, path: props.path, popout });
  }, [ onClick, popout, props.path, external ]);

  return props.disabled ? (
    <span className={classes.join(' ')}>{props.children}</span>
  ) : (
    <a
      aria-label={props.label}
      className={classes.join(' ')}
      href={href}
      rel={rel}
      onClick={!props.rawLink ? handleClick : undefined}>
      {props.children}
    </a>
  );
};

export default Link;
