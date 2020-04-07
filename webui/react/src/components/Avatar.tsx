import React from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import { hex2hsl, hsl2str } from 'utils/color';
import md5 from 'utils/md5';

interface Props {
  name: string;
}

const Avatar: React.FC<Props> = (props: Props) => {
  const getInitials = (name: string): string => {
    // Reduce the name to initials.
    const initials = name
      .split(/\s+/)
      .map(n => n.charAt(0).toUpperCase())
      .join('');

    // If initials are long, just keep the first and the last.
    return initials.length > 2 ? `${initials.charAt(0)}${initials.substr(-1)}` : initials;
  };

  const getColor = (name: string): string => {
    const hexColor = md5(name).substr(0, 6);
    const hslColor = hex2hsl(hexColor);
    return hsl2str({ ...hslColor, l: 50 });
  };

  return (
    <Base color={getColor(props.name)} id="avatar">{getInitials(props.name)}</Base>
  );
};

const Base = styled.div<{ color: string }>`
  align-items: center;
  background-color: ${(props): string => props.color};
  border-radius: 100%;
  color: white;
  display: flex;
  font-size: 1rem;
  font-weight: bold;
  height: ${theme('sizes.layout.jumbo')};
  justify-content: center;
  width: ${theme('sizes.layout.jumbo')};
`;

export default Avatar;
