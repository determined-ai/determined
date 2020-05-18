import React, { PropsWithChildren, useCallback } from 'react';

import { routeAll, setupUrlForDev } from 'routes';

import css from './Link.module.scss';

interface Props {
  disabled?: boolean;
  path: string;
  popout?: boolean;
  onClick?: (event: React.MouseEvent) => void;
}

const Link: React.FC<Props> = ({
  disabled, path, popout, onClick, children,
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];

  if (!disabled) classes.push(css.link);

  const handleClick = useCallback((event: React.MouseEvent): void => {
    const url = setupUrlForDev(path);

    event.persist();
    event.preventDefault();

    if (onClick) {
      onClick(event);
    } else if (event.metaKey || event.ctrlKey || popout) {
      window.open(url, '_blank');
    } else {
      routeAll(url);
    }
  }, [ onClick, path, popout ]);

  return <a className={classes.join(' ')} href={path} onClick={handleClick}>{children}</a>;
};

export default Link;
