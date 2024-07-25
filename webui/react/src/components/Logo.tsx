import React, { useMemo } from 'react';

import logoDeterminedOnDarkHorizontal from 'assets/images/logo-determined-on-dark-horizontal.svg?url';
import logoDeterminedOnDarkVertical from 'assets/images/logo-determined-on-dark-vertical.svg?url';
import logoDeterminedOnLightHorizontal from 'assets/images/logo-determined-on-light-horizontal.svg?url';
import logoDeterminedOnLightVertical from 'assets/images/logo-determined-on-light-vertical.svg?url';
import logoHpeOnDarkHorizontal from 'assets/images/logo-hpe-on-dark-horizontal.svg?url';
import logoHpeOnLightHorizontal from 'assets/images/logo-hpe-on-light-horizontal.svg?url';
import useUI from 'components/ThemeProvider';
import { linkPath, paths, serverAddress } from 'routes/utils';
import { BrandingType } from 'stores/determinedInfo';
import { ValueOf } from 'types';
import { reactHostAddress } from 'utils/routes';

import css from './Logo.module.scss';

const Orientation = {
  Horizontal: 'horizontal',
  Vertical: 'vertical',
} as const;

type Orientation = ValueOf<typeof Orientation>;

interface Props {
  branding?: BrandingType;
  orientation?: Orientation;
  hasCustomLogo?: boolean;
}

const logos: Record<BrandingType, Record<Orientation, Record<string, string>>> = {
  [BrandingType.Determined]: {
    [Orientation.Horizontal]: {
      dark: logoDeterminedOnDarkHorizontal,
      light: logoDeterminedOnLightHorizontal,
    },
    [Orientation.Vertical]: {
      dark: logoDeterminedOnDarkVertical,
      light: logoDeterminedOnLightVertical,
    },
  },
  [BrandingType.HPE]: {
    [Orientation.Horizontal]: {
      dark: logoHpeOnDarkHorizontal,
      light: logoHpeOnLightHorizontal,
    },
    [Orientation.Vertical]: {
      dark: logoHpeOnDarkHorizontal,
      light: logoHpeOnLightHorizontal,
    },
  },
};

const Logo: React.FC<Props> = ({
  branding = BrandingType.Determined,
  orientation = Orientation.Vertical,
  hasCustomLogo = false,
}: Props) => {
  const { isDarkMode } = useUI();

  const alt = useMemo(() => {
    const isDetermined = branding === BrandingType.Determined;
    const server = serverAddress();
    const isSameServer = reactHostAddress() === server;
    return [
      isDetermined ? 'Determined AI Logo' : 'HPE Machine Learning Development Logo',
      isSameServer ? '' : ` (Server: ${server})`,
    ].join();
  }, [branding]);

  const mode = isDarkMode ? 'dark' : 'light';
  const imageSrc = hasCustomLogo
    ? linkPath(paths.customerAsset('logo', orientation, mode), false)
    : logos[branding][orientation][mode];
  return <img alt={alt} className={css.logo} src={imageSrc} />;
};

export default Logo;
