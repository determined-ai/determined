import React, { MouseEvent, MouseEventHandler, PropsWithChildren, useCallback } from 'react';

import { handlePath, windowOpenFeatures } from 'utils/routes';

import css from './Link.module.scss';

interface Props {
  className?: string;
  disabled?: boolean;
  inherit?: boolean;
  isButton?: boolean;
  path?: string;
  popout?: boolean;
  onClick?: MouseEventHandler;
}

const Link: React.FC<Props> = ({
  path = '#',
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

  const handleClick = useCallback((event: MouseEvent) => {
    handlePath(event, { onClick, path, popout });
  }, [ onClick, path, popout ]);

  return props.disabled ? (
    <span className={classes.join(' ')}>{props.children}</span>
  ) : (
    <a
      className={classes.join(' ')}
      href={path}
      rel={rel}
      onClick={handleClick}>
      {props.children}
    </a>
  );
};

export default Link;
