import React, { MouseEventHandler, PropsWithChildren, useCallback } from 'react';

import { routeAll, setupUrlForDev } from 'routes';
import { openBlank, windowOpenFeatures } from 'utils/routes';

import css from './Link.module.scss';

interface Props {
  disabled?: boolean;
  inherit?: boolean;
  path: string;
  popout?: boolean;
  onClick?: MouseEventHandler;
}

export const makeClickHandler = (
  path: string,
  onClick?: MouseEventHandler,
  popout?: boolean,
): MouseEventHandler => {
  const handler: MouseEventHandler = (event) => {
    const url = setupUrlForDev(path);

    event.persist();
    event.preventDefault();

    if (onClick) {
      onClick(event);
    } else if (event.metaKey || event.ctrlKey || popout) {
      openBlank(url);
    } else {
      routeAll(url);
    }
  };
  return handler;
};

const Link: React.FC<Props> = ({
  disabled, inherit, path, popout, onClick, children,
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const rel = windowOpenFeatures.join(' ');
  const handleClick =
    useCallback(makeClickHandler(path, onClick, popout), [ path, onClick, popout ]);

  if (!disabled) classes.push(css.link);
  if (inherit) classes.push(css.inherit);

  if (disabled) {
    return <span className={classes.join(' ')}>
      {children}
    </span>;
  }

  return (
    <a className={classes.join(' ')} href={path} rel={rel}
      onClick={handleClick}>
      {children}
    </a>
  );
};

export default Link;
