import React from 'react';

import useUI from 'shared/contexts/stores/UI';
import { DarkLight, getCssVar } from 'shared/themes';
import { Status } from 'utils/colors';

const SVG: React.FC<{ color: string }> = ({ color }) => (
  <svg fill="none" height="24" viewBox="0 0 24 24" width="24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M10.72 5H19V19H10.72V5ZM9.72 5H5V19H9.72V5ZM10.72 4H19C19.5523 4 20 4.44772 20 5V19C20 19.5523 19.5523 20 19 20H10.72H9.72H5C4.93096 20 4.86356 19.993 4.79847 19.9797C4.34278 19.8864 4 19.4832 4 19V5C4 4.44772 4.44772 4 5 4H9.72H10.72Z"
      fill={color}
      fillRule="evenodd"
    />
  </svg>
);

const PanelIcon: React.FC = () => {
  const {
    ui: { darkLight },
  } = useUI();

  return (
    <SVG
      color={getCssVar(darkLight === DarkLight.Dark ? Status.InactiveStrong : Status.Inactive)}
    />
  );
};

export default PanelIcon;
