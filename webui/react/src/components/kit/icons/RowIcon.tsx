import React from 'react';

import useUI from 'shared/contexts/stores/UI';
import { DarkLight, getCssVar } from 'shared/themes';
import { Status } from 'utils/colors';

type Size = 'small' | 'medium' | 'large' | 'xl';

type Props = { size?: Size };

const SVGLarge: React.FC<{ color: string }> = ({ color }) => (
  <svg fill="none" height="24" viewBox="0 0 24 24" width="24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M5 6H19V14H5V6ZM4 6C4 5.44772 4.44772 5 5 5H19C19.5523 5 20 5.44772 20 6V14C20 14.5523 19.5523 15 19 15H5C4.44772 15 4 14.5523 4 14V6ZM4.5 18C4.22386 18 4 18.2239 4 18.5C4 18.7761 4.22386 19 4.5 19H19.5C19.7761 19 20 18.7761 20 18.5C20 18.2239 19.7761 18 19.5 18H4.5Z"
      fill={color}
      fillRule="evenodd"
    />
  </svg>
);
const SVGMedium: React.FC<{ color: string }> = ({ color }) => (
  <svg fill="none" height="24" viewBox="0 0 24 24" width="24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M19 6H5V10H19V6ZM5 5C4.44772 5 4 5.44772 4 6V10C4 10.5523 4.44772 11 5 11H19C19.5523 11 20 10.5523 20 10V6C20 5.44772 19.5523 5 19 5H5ZM4 14.5C4 14.2239 4.22386 14 4.5 14H19.5C19.7761 14 20 14.2239 20 14.5C20 14.7761 19.7761 15 19.5 15H4.5C4.22386 15 4 14.7761 4 14.5ZM4.5 18C4.22386 18 4 18.2239 4 18.5C4 18.7761 4.22386 19 4.5 19H19.5C19.7761 19 20 18.7761 20 18.5C20 18.2239 19.7761 18 19.5 18H4.5Z"
      fill={color}
      fillRule="evenodd"
    />
  </svg>
);
const SVGExtraLarge: React.FC<{ color: string }> = ({ color }) => (
  <svg fill="none" height="24" viewBox="0 0 24 24" width="24" xmlns="http://www.w3.org/2000/svg">
    <rect height="13" rx="0.5" stroke={color} width="15" x="4.5" y="5.5" />
  </svg>
);
const SVGSmall: React.FC<{ color: string }> = ({ color }) => (
  <svg fill="none" height="24" viewBox="0 0 24 24" width="24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M4.5 6C4.22386 6 4 6.22386 4 6.5C4 6.77614 4.22386 7 4.5 7H19.5C19.7761 7 20 6.77614 20 6.5C20 6.22386 19.7761 6 19.5 6H4.5ZM4.5 10C4.22386 10 4 10.2239 4 10.5C4 10.7761 4.22386 11 4.5 11H19.5C19.7761 11 20 10.7761 20 10.5C20 10.2239 19.7761 10 19.5 10H4.5ZM4 14.5C4 14.2239 4.22386 14 4.5 14H19.5C19.7761 14 20 14.2239 20 14.5C20 14.7761 19.7761 15 19.5 15H4.5C4.22386 15 4 14.7761 4 14.5ZM4.5 18C4.22386 18 4 18.2239 4 18.5C4 18.7761 4.22386 19 4.5 19H19.5C19.7761 19 20 18.7761 20 18.5C20 18.2239 19.7761 18 19.5 18H4.5Z"
      fill={color}
      fillRule="evenodd"
    />
  </svg>
);

const RowIcon: React.FC<Props> = ({ size = 'medium' }) => {
  const {
    ui: { darkLight },
  } = useUI();

  const icon = React.useMemo(() => {
    const color = getCssVar(darkLight === DarkLight.Dark ? Status.InactiveStrong : Status.Inactive);
    const sizes = {
      large: <SVGLarge color={color} />,
      medium: <SVGMedium color={color} />,
      small: <SVGSmall color={color} />,
      xl: <SVGExtraLarge color={color} />,
    };

    return sizes[size];
  }, [size, darkLight]);

  return <>{icon}</>;
};

export default RowIcon;
