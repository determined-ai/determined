import React from 'react';

import css from 'components/Image/Image.module.scss';
import { DarkLight } from 'utils/themes';

export interface Props {
  darkLight?: DarkLight;
}

export const ImageAlert: React.FC<Props> = ({ darkLight }) => {
  const classes = [css.alert];
  if (darkLight === DarkLight.Dark) classes.push(css.dark);
  return (
    <svg
      className={classes.join(' ')}
      fill="none"
      height="100"
      viewBox="0 0 1024 1024"
      width="100"
      xmlns="http://www.w3.org/2000/svg">
      <title>Alert</title>
      <ellipse className={css.shadow1} cx="512" cy="844" rx="400" ry="60" />
      <ellipse className={css.shadow2} cx="512" cy="844" rx="300" ry="40" />
      <ellipse className={css.shadow3} cx="502" cy="844" rx="200" ry="20" />
      <circle className={css.sign} cx="512" cy="402" fill="white" r="265" strokeWidth="10" />
      <circle className={css.iDot} r="10" transform="matrix(1 0 0 -1 512 277)" />
      <rect
        className={css.iBar}
        height="200"
        rx="5"
        transform="matrix(1 0 0 -1 507 537)"
        width="10"
      />
    </svg>
  );
};

export const ImageEmpty: React.FC<Props> = () => (
  <svg
    className="ant-empty-img-simple"
    height="100"
    viewBox="-8 -5 80 51"
    width="100"
    xmlns="http://www.w3.org/2000/svg">
    <title>Empty</title>
    <g fill="none" fillRule="evenodd" transform="translate(0 1)">
      <ellipse className="ant-empty-img-simple-ellipse" cx="32" cy="33" rx="32" ry="7" />
      <g className="ant-empty-img-simple-g" fillRule="nonzero">
        <path
          d="M55 12.76L44.854 1.258C44.367.474 43.656 0 42.907
            0H21.093c-.749 0-1.46.474-1.947 1.257L9 12.761V22h46v-9.24z"
        />
        <path
          className="ant-empty-img-simple-path"
          d="M41.613 15.931c0-1.605.994-2.93 2.227-2.931H55v18.137C55 33.26 53.68 35 52.05
            35h-40.1C10.32 35 9 33.259 9 31.137V13h11.16c1.233 0 2.227 1.323 2.227 2.928v.022c0
            1.605 1.005 2.901 2.237 2.901h14.752c1.232 0 2.237-1.308 2.237-2.913v-.007z"
        />
      </g>
    </g>
  </svg>
);

export const ImageWarning: React.FC<Props> = ({ darkLight }) => {
  const classes = [css.warning];
  if (darkLight === DarkLight.Dark) classes.push(css.dark);
  return (
    <svg
      className={classes.join(' ')}
      fill="none"
      height="100"
      viewBox="0 0 1024 1024"
      width="100"
      xmlns="http://www.w3.org/2000/svg">
      <title>Warning</title>
      <ellipse className={css.shadow1} cx="512" cy="844" rx="400" ry="60" />
      <ellipse className={css.shadow2} cx="512" cy="844" rx="300" ry="40" />
      <ellipse className={css.shadow3} cx="502" cy="844" rx="200" ry="20" />
      <path
        className={css.sign}
        d="M477.874 175.814L228.215 584.134C211.918 610.788 231.1 645 262.342
        645H761.658C792.9 645 812.082 610.788 795.785 584.134L546.126 175.814C530.528
        150.302 493.472 150.302 477.874 175.814Z"
        strokeLinejoin="round"
        strokeWidth="10"
      />
      <circle className={css.exclamationDot} cx="512" cy="551" r="10" />
      <rect className={css.exclamationBar} height="200" rx="5" width="10" x="507" y="291" />
    </svg>
  );
};
