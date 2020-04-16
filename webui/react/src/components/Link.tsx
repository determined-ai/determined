import React, { PropsWithChildren, useCallback } from 'react';
import { useHistory } from 'react-router-dom';

import css from './Link.module.scss';

interface Props {
  crossover?: boolean;
  disabled?: boolean;
  path: string;
  popout?: boolean;
  onClick?: (event: React.MouseEvent) => void;
}

const defaultProps = {
  crossover: false,
};

const Link: React.FC<Props> = ({
  crossover, disabled, path, popout, onClick, children,
}: PropsWithChildren<Props>) => {
  const history = useHistory();
  const classes = [];

  if (!disabled) classes.push(css.link);

  const handleClick = useCallback((event: React.MouseEvent): void => {
    const pathPrefix = process.env.IS_DEV ? 'http://localhost:8080' : '';
    const url = crossover ? `${pathPrefix}${path}` : path;

    event.persist();
    event.preventDefault();

    if (onClick) {
      onClick(event);
    } else if (event.metaKey || event.ctrlKey || popout) {
      window.open(url, '_blank');
    } else if (crossover) {
      window.location.assign(url);
    } else {
      history.push(url);
    }
  }, [ history, crossover, onClick, path, popout ]);

  return <div className={classes.join(' ')} onClick={handleClick}>{children}</div>;
};

Link.defaultProps = defaultProps;

export default Link;
