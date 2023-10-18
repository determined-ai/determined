import React, { MouseEvent, useCallback } from 'react';

import KitLink from 'components/kit/Link';
import { handlePath, linkPath } from 'routes/utils';
import { AnyMouseEventHandler, windowOpenFeatures } from 'utils/routes';

export interface Props {
  children?: React.ReactNode;
  disabled?: boolean;
  // is this external to the assets hosted by React?
  external?: boolean;
  onClick?: AnyMouseEventHandler;
  path?: string;
  popout?: boolean | 'tab' | 'window';
  size?: 'tiny' | 'small' | 'medium' | 'large';
}

const Link: React.FC<Props> = ({ external, popout, onClick, ...props }: Props) => {
  const rel = windowOpenFeatures.join(' ');

  const href = props.path ? linkPath(props.path, external) : undefined;
  const handleClick = useCallback(
    (event: MouseEvent) => {
      handlePath(event, { external, onClick, path: props.path, popout });
    },
    [onClick, popout, props.path, external],
  );

  return (
    <KitLink
      disabled={props.disabled}
      href={href}
      rel={rel}
      size={props.size}
      onClick={handleClick}>
      {props.children}
    </KitLink>
  );
};

export default Link;
