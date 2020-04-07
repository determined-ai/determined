import React from 'react';
import { useHistory } from 'react-router-dom';
import styled from 'styled-components';

interface Props {
  crossover?: boolean;
  disabled?: boolean;
  path: string;
  popout?: boolean;
  onClick?: (event: React.MouseEvent) => void;
  children: React.ReactNode;
}

const defaultProps = {
  crossover: false,
};

const Link: React.FC<Props> = (props: Props) => {
  const history = useHistory();

  if (props.disabled) return <div {...props}>{props.children}</div>;

  const handleClick = (event: React.MouseEvent): void => {
    const pathPrefix = process.env.IS_DEV ? 'http://localhost:8080' : '';
    const url = props.crossover ? `${pathPrefix}${props.path}` : props.path;

    event.persist();
    event.preventDefault();

    if (props.onClick) {
      props.onClick(event);
    } else if (event.metaKey || event.ctrlKey || props.popout) {
      window.open(url, '_blank');
    } else if (props.crossover) {
      window.location.assign(url);
    } else {
      history.push(url);
    }
  };

  return <Base {...props} onClick={handleClick}>{props.children}</Base>;
};

const Base = styled.a`
  cursor: pointer;
`;

Link.defaultProps = defaultProps;

export default Link;
