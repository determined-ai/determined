import React, { PropsWithChildren, useCallback } from 'react';

import { routeAll, setupUrlForDev } from 'routes';

import css from './Link.module.scss';

type OnClick = (event: React.MouseEvent) => void

interface Props {
  disabled?: boolean;
  inherit?: boolean;
  path: string;
  popout?: boolean;
  onClick?: OnClick;
}

const windowFeatures = [ 'noopener', 'noreferrer' ];

export const handleClick = (path: string, onClick?: OnClick,  popout?: boolean): OnClick => {
  return (event: React.MouseEvent): void => {
    const url = setupUrlForDev(path);

    event.persist();
    event.preventDefault();

    if (onClick) {
      onClick(event);
    } else if (event.metaKey || event.ctrlKey || popout) {
      window.open(url, '_blank', windowFeatures.join(','));
    } else {
      routeAll(url);
    }
  };
};

const Link: React.FC<Props> = ({
  disabled, inherit, path, popout, onClick, children,
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const rel = windowFeatures.join(' ');

  if (!disabled) classes.push(css.link);
  if (inherit) classes.push(css.inherit);

  return (
    <a className={classes.join(' ')} href={path} rel={rel}
      onClick={handleClick(path, onClick, popout)}>
      {children}
    </a>
  );
};

export default Link;
