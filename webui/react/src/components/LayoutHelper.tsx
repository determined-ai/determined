import React, { PropsWithChildren } from 'react';
import styled, { css, FlattenSimpleInterpolation } from 'styled-components';
import { ifProp } from 'styled-tools';

import { PropsWithTheme, ShirtSize } from 'themes';
import { isPropTrue } from 'utils/styled';

interface Props {
  center?: boolean; //  == xCenter + yCenter
  column?: boolean;
  fullHeight?: boolean;
  fullWidth?: boolean;
  gap?: ShirtSize;
  grow?: boolean;
  padding?: ShirtSize[];
  paddingBottom?: ShirtSize;
  paddingLeft?: ShirtSize;
  paddingRight?: ShirtSize;
  paddingTop?: ShirtSize;
  spaceBetween?: boolean;
  xCenter?: boolean;
  xEnd?: boolean;
  xStart?: boolean;
  yCenter?: boolean;
  yEnd?: boolean;
  yStart?: boolean;
}

const xProps = new Set([ 'xStart', 'xCenter', 'xEnd' ]);
const yProps = new Set([ 'yStart', 'yCenter', 'yEnd' ]);

const LayoutHelper: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const checkUniqueAttributes = (props: Props): void => {
    const attrs = Object.keys(props);
    if (attrs.filter(prop => xProps.has(prop) && isPropTrue(props, prop)).length > 1)
      console.warn('can not have more than one of "xStart", "xCenter" and "xEnd" props');
    if (attrs.filter(prop => yProps.has(prop) && isPropTrue(props, prop)).length > 1)
      console.warn('can not have more than one of "yStart", "yCenter" and "yEnd" props');
    if (props.center &&
      attrs.some(prop => (xProps.has(prop) || yProps.has(prop)) && isPropTrue(props, prop)))
      console.warn('when using "center" prop, the "x" and "y" props should not be used');
  };

  // We do not ensure that the attributes do not collide. Instead we will pick the first one that
  // we run into as discussed in PR#4790.
  checkUniqueAttributes(props);

  return (
    <Base {...props}>{props.children}</Base>
  );
};

const getPlacementStyles = (props: Props): string => {
  const styles = [];

  const alignStyles: Record<string, FlattenSimpleInterpolation> = {
    center: css`align-items: center;`,
    end: css`align-items: flex-end;`,
    start: css`align-items: flex-start;`,
  };

  const justifyStyles: Record<string, FlattenSimpleInterpolation> = {
    center: css`justify-content: center;`,
    end: css`justify-content: flex-end;`,
    spaceBetween: css`justify-content: space-between;`,
    start: css`justify-content: flex-start;`,
  };

  const [ xStyles, yStyles ] = props.column ?
    [ alignStyles, justifyStyles ] : [ justifyStyles, alignStyles ];

  const propToAttrName = (prop: string): string => prop.slice(1).toLowerCase();

  const xProp = Object.keys(props).find(p => xProps.has(p) && isPropTrue(props, p));
  if (xProp) {
    styles.push(xStyles[propToAttrName(xProp)]);
  }
  const yProp = Object.keys(props).find(p => yProps.has(p) && isPropTrue(props, p));
  if (yProp) {
    styles.push(yStyles[propToAttrName(yProp)]);
  }
  if (props.center) {
    styles.push(xStyles.center);
    styles.push(yStyles.center);
  }

  if (styles.length === 0) {
    styles.push(xStyles.start, yStyles.start);
  }

  if (props.spaceBetween) {
    styles.push(justifyStyles.spaceBetween);
  }

  return styles.map(css => `${css}`).join('');
};

const getSinglePadding = (props: PropsWithTheme<Props>): string => {
  if (!props.padding) return '';
  const paddingValue = props.padding
    .map(shirtSize => props.theme.sizes.layout[shirtSize])
    .join(' ');
  return `padding: ${paddingValue};`;
};

const getIndividualPaddings = (props: PropsWithTheme<Props>): string => {
  const layoutTheme = props.theme.sizes.layout;
  return [
    props.paddingBottom ? `padding-bottom: ${layoutTheme[props.paddingBottom]};` : '',
    props.paddingLeft ? `padding-left: ${layoutTheme[props.paddingLeft]};` : '',
    props.paddingRight ? `padding-right: ${layoutTheme[props.paddingRight]};` : '',
    props.paddingTop ? `padding-top: ${layoutTheme[props.paddingTop]};` : '',
  ].join('');
};

const getGap = (props: PropsWithTheme<Props>): string => {
  if (!props.gap) return '';
  return `
    & > *:not(:first-child) {
      margin-${props.column ? 'top' : 'left'}: ${props.theme.sizes.layout[props.gap]};
    }
  `;
};

const Base = styled.div<Props>`
  display: flex;
  flex-direction: ${ifProp('column', 'column', 'row')};
  overflow: hidden;
  ${getGap}
  ${ifProp('fullHeight', 'height: 100%;')}
  ${ifProp('fullWidth', 'width: 100%;')}
  ${getSinglePadding}
  ${getIndividualPaddings}
  ${getPlacementStyles}
  ${ifProp('grow', '> * { flex-grow: 1; }', '')}
`;

export default LayoutHelper;
