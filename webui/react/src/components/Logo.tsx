import React from 'react';

import logoSource from 'assets/logo-on-dark-horizontal.svg';

import css from './Logo.module.scss';

const Logo: React.FC = () => {
  return <img alt="Determined AI Logo" className={css.base} src={logoSource} />;
};

export default Logo;
