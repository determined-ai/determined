import React from 'react';

import Icon from 'components/Icon';

import css from './Spinner.module.scss';

interface Props {
  fullPage?: boolean;
}

const defaultProps: Props = { fullPage: false };

const Spinner: React.FC<Props> = ({ fullPage }: Props) => {
  const classes = [ css.base ];

  if (fullPage) classes.push(css.fullPage);

  return (
    <div className={classes.join(' ')}>
      <div className={css.spin}>
        <Icon name="spinner" size="large" />
      </div>
    </div>
  );
};

Spinner.defaultProps = defaultProps;

export default Spinner;
