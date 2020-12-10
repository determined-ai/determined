import React, { MouseEvent, MouseEventHandler, PropsWithChildren, useCallback } from 'react';

import { handlePath, linkPath, windowOpenFeatures } from 'routes/utils';

import css from './Link.module.scss';

export interface Props {
  className?: string;
  disabled?: boolean;
  // is this external to the assets hosted by React?
  external?: boolean;
  inherit?: boolean;
  isButton?: boolean;
  label?: string;
  onClick?: MouseEventHandler;
  path?: string;
  popout?: boolean;
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
  if (!props.disabled) classes.push(css.link);
  if (props.inherit) classes.push(css.inherit);
  if (props.isButton) classes.push('ant-btn');

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
      onClick={handleClick}>
      {props.children}
    </a>
  );
};

export default Link;
