import React, { MouseEvent, MouseEventHandler, PropsWithChildren, useCallback } from 'react';

import { handlePath, serverAddress, windowOpenFeatures } from 'utils/routes';

import css from './Link.module.scss';

export interface LinkProps {
  className?: string;
  disabled?: boolean;
  inherit?: boolean;
  isButton?: boolean;
  path?: string;
  popout?: boolean;
  proxy?: boolean;
  onClick?: MouseEventHandler;
}

const Link: React.FC<LinkProps> = ({
  path = '#',
  popout,
  proxy,
  onClick,
  ...props
}: PropsWithChildren<LinkProps>) => {
  const classes = [ css.base ];
  const rel = windowOpenFeatures.join(' ');
  const proxyPath = `${proxy ? serverAddress(true) : ''}${path}`;

  if (props.className) classes.push(props.className);
  if (!props.disabled) classes.push(css.link);
  if (props.inherit) classes.push(css.inherit);
  if (props.isButton) classes.push('ant-btn');

  const handleClick = useCallback((event: MouseEvent) => {
    handlePath(event, { onClick, path: proxyPath, popout });
  }, [ onClick, popout, proxyPath ]);

  return props.disabled ? (
    <span className={classes.join(' ')}>{props.children}</span>
  ) : (
    <a
      className={classes.join(' ')}
      href={proxyPath}
      rel={rel}
      onClick={handleClick}>
      {props.children}
    </a>
  );
};

export default Link;
