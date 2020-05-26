import { HomeOutlined } from '@ant-design/icons';
import { Breadcrumb as ABC } from 'antd';
import React from 'react';

import Icon from 'components/Icon';
import Link from 'components/Link';
import { RouteConfigItem } from 'routes';
import { isAbsolutePath } from 'utils/routes';

import css from './Breadcrumb.module.scss';

interface Props {
  route: RouteConfigItem;
}

type ComputedText = (arg0: string) => string

interface Section {
  icon?: React.ReactNode;
  text?: string | ComputedText;
}

// should be truned into an ordered list if overlaping entries are added
const partToSection: Record<string, Section> = {
  '/[0-9]+/': { text: (id: string): string => id.toString() },
  'dashboard': { icon: <Icon name="user" size="small" />, text: 'Dashboard' },
  'det': { icon: <HomeOutlined /> },
  'experiments': { icon: <Icon name="experiment" size="small" />, text: 'Experiments' },
  'trials': { icon: <Icon name="user" />, text: 'Trials' },
};

const getSection = (part: string): Section => {
  // use regex or router match to find the matching provider

  // we can bring in and user appRoutes and try to avoid rebuilding parts of the routes
  // here but we do want to avoid having breadcrumbs dictate how routes are structured.
  // const route = appRoutes.find(route => route.id === part);
  const key = Object.keys(partToSection).find(key => !!part.match(key));
  return key ? partToSection[key] : { text: (part: string) => part } ;
};

const pathToABCItems = (path: string): React.ReactNode => {
  if (!isAbsolutePath(path)) throw new Error('path needs to be absolute');
  const parts = path.substring(1).split('/');
  return parts.map((part, idx) => {
    const section = getSection(part);
    let text;
    if (typeof section.text === 'string') text = section.text;
    if (typeof section.text === 'function') text = section.text(part);

    return (
      <ABC.Item key={idx}>
        <Link path={'/' + parts.slice(0,idx+1).join('/')}>
          {section.icon && !section.text && section.icon}
          {section.text && !section.icon && <span>{text}</span>}
          {section.text && section.icon && <span className={css.full}>{section.icon} {text}</span>}
        </Link>
      </ABC.Item>
    );
  });
};

const Breadcrumb: React.FC<Props> = ({ route }: Props) => {
  return (
    <ABC className={css.base}>
      {pathToABCItems(route.path)}
    </ABC>
  );
};

export default Breadcrumb;
