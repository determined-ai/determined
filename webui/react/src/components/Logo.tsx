import React from 'react';
import styled from 'styled-components';

import logoSource from 'assets/logo-on-dark-horizontal.svg';

const Base = styled.img`
  height: 2rem;
  width: 12.8rem;
`;

const Logo: React.FC = () => {
  return <Base alt="Determined AI Logo" src={logoSource} />;
};

export default Logo;
