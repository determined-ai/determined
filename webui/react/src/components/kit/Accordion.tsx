import { Collapse } from 'antd';
import React from 'react';

import { ConditionalWrapper } from 'components/kit/internal/ConditionalWrapper';

interface AccordionProps {
  title: React.ReactNode;
  children: React.ReactNode;
  key?: string | number;
  open?: boolean;
  defaultOpen?: boolean;
  mountChildren?: 'immediately' | 'on-open' | 'once-on-open';
}

interface AccordionGroupProps {
  exclusive?: boolean;
  openKey?: string | number | string[] | number[];
  defaultOpenKey?: string | number | string[] | number[];
  children: React.ReactNode;
}

const AccordionGroup: React.FC<AccordionGroupProps> = ({
  exclusive,
  children,
  openKey,
  defaultOpenKey,
}) => {
  // handle uncontrolled behavior
  const activeProp = openKey !== undefined ? { activeKey: openKey } : {};
  // disable default collapse accordion behavior where the first child is open
  const defaultActiveKey = defaultOpenKey ? defaultOpenKey : exclusive ? [] : undefined;
  return (
    <Collapse accordion={exclusive} {...activeProp} defaultActiveKey={defaultActiveKey}>
      {children}
    </Collapse>
  );
};

const Accordion: React.FC<AccordionProps> & { Group: typeof AccordionGroup } = ({
  title,
  children,
  open,
  defaultOpen,
  mountChildren = 'once-on-open',
  ...otherProps
}) => {
  // Collapse passes housekeeping props through to the child -- assume
  // that we're in a multi-accordion situation when we encounter this
  const inGroup = 'panelKey' in otherProps;
  // key used for single accordion cases
  const key = -1;

  const wrapper = (c: React.ReactElement) => {
    // handle isActive for single accordions
    let collapseProps = {};
    if (defaultOpen !== undefined) {
      collapseProps = { defaultActiveKey: defaultOpen ? key : '' };
    }
    if (open !== undefined) {
      collapseProps = { activeKey: open ? key : '' };
    }
    return <Collapse {...collapseProps}>{c}</Collapse>;
  };
  let panelProps = {};
  switch (mountChildren) {
    case 'immediately': {
      panelProps = { forceRender: true };
      break;
    }
    case 'on-open': {
      panelProps = { destroyInactivePanel: true };
      break;
    }
    case 'once-on-open':
    default: {
      panelProps = {};
    }
  }

  return (
    <ConditionalWrapper condition={!inGroup} wrapper={wrapper}>
      <Collapse.Panel header={title} {...otherProps} {...panelProps} key={key}>
        {children}
      </Collapse.Panel>
    </ConditionalWrapper>
  );
};

export default Accordion;
Accordion.Group = AccordionGroup;
