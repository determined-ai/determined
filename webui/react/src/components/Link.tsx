import { inherits } from 'util';

import React, { PropsWithChildren, useCallback } from 'react';

import { routeAll, setupUrlForDev } from 'routes';

import css from './Link.module.scss';

interface Props {
  disabled?: boolean;
  inherit?: boolean;
  path: string;
  popout?: boolean;
  onClick?: (event: React.MouseEvent) => void;
}

const windowFeatures = [ 'noopener', 'noreferrer' ];

const Link: React.FC<Props> = ({
  disabled, inherit, path, popout, onClick, children,
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const rel = windowFeatures.join(' ');

  if (!disabled) classes.push(css.link);
  if (inherit) classes.push(css.inherit);

  const handleClick = useCallback((event: React.MouseEvent): void => {
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
  }, [ onClick, path, popout ]);

  return (
    <a className={classes.join(' ')} href={path} rel={rel} onClick={handleClick}>
      {children}
    </a>
  );
};

export default Link;
