import { StyleProvider } from '@ant-design/cssinjs';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import Accordion from './Accordion';

const user = userEvent.setup();
type AccordionProps = Parameters<typeof Accordion>[0];
type AccordionGroupProps = Parameters<typeof Accordion.Group>[0];

const titleText = 'title';
const childText = 'child';
const singleSetup = (props: Omit<AccordionProps, 'title' | 'children'> = {}) => {
  return render(
    <StyleProvider container={document.body} hashPriority="high">
      <Accordion title={titleText} {...props}>
        {childText}
      </Accordion>
    </StyleProvider>,
  );
};

const groupInfo = {
  [1]: {
    child: 'One -- Child',
    title: 'One -- Title',
  },
  [2]: {
    child: 'Two -- Child',
    title: 'Two -- Title',
  },
  [3]: {
    child: 'Three -- Child',
    title: 'Three -- Title',
  },
} as const;

const groupSetup = (props: Omit<AccordionGroupProps, 'children'> = {}) => {
  return render(
    <StyleProvider container={document.body} hashPriority="high">
      <Accordion.Group {...props}>
        {Object.entries(groupInfo).map(([key, texts]) => (
          // children are mounted immediately to make testing the group state easier
          <Accordion key={key} mountChildren="immediately" title={texts.title}>
            {texts.child}
          </Accordion>
        ))}
      </Accordion.Group>
    </StyleProvider>,
  );
};

describe('Accordion', () => {
  describe('single', () => {
    it('displays title and children when clicked', async () => {
      singleSetup();
      const title = screen.getByText(titleText);
      expect(title).toBeVisible();
      await user.click(title);
      expect(screen.getByText(childText)).toBeVisible();
    });
    describe('mountChildren', () => {
      it('when mountChildren is set to immediately, the child is mounted on mount', () => {
        singleSetup({ mountChildren: 'immediately' });
        expect(screen.getByText(childText)).toBeInTheDocument();
      });
      it('when mountChildren is set to on-mount, the child is unmounted on close', async () => {
        singleSetup({ mountChildren: 'on-open' });
        expect(screen.queryByText(childText)).toBeNull();
        const title = screen.getByText(titleText);
        await user.click(title);
        expect(screen.getByText(childText)).toBeInTheDocument();
        await user.click(title);
        expect(screen.queryByText(childText)).toBeNull();
      });
      it('when mountchildren is set to on-mount-once, the child remains mounted on close', async () => {
        singleSetup({ mountChildren: 'once-on-open' });
        expect(screen.queryByText(childText)).toBeNull();
        const title = screen.getByText(titleText);
        await user.click(title);
        expect(screen.getByText(childText)).toBeInTheDocument();
        await user.click(title);
        expect(screen.getByText(childText)).toBeInTheDocument();
      });
    });
    describe('control', () => {
      it('when defaultOpen is set to true, the child is open on mount and uncontrolled', async () => {
        singleSetup({ defaultOpen: true });
        const child = screen.getByText(childText);
        const title = screen.getByText(titleText);
        expect(child).toBeVisible();
        await user.click(title);
        expect(child).not.toBeVisible();
      });
      it('when defaultOpen is set to false, the child is closed on mount and uncontrolled', async () => {
        singleSetup({ defaultOpen: false });
        expect(screen.queryByText(childText)).toBeNull();
        const title = screen.getByText(titleText);
        await user.click(title);
        expect(screen.queryByText(childText)).toBeVisible();
      });
      it('when open is true, the child is open on mount and controlled', async () => {
        singleSetup({ open: true });
        const child = screen.getByText(childText);
        const title = screen.getByText(titleText);
        expect(child).toBeVisible();
        await user.click(title);
        expect(child).toBeVisible();
      });
      it('when open is false, the child is closed on mount and controlled', async () => {
        singleSetup({ open: false });
        expect(screen.queryByText(childText)).toBeNull();
        const title = screen.getByText(titleText);
        await user.click(title);
        expect(screen.queryByText(childText)).toBeNull();
      });
    });
  });
  describe('group', () => {
    type AccordionState = 'open' | 'closed';
    interface GroupState {
      [1]: AccordionState;
      [2]: AccordionState;
      [3]: AccordionState;
    }
    const expectGroupState = (groupState: GroupState) => {
      Object.entries(groupState).forEach(([key, val]) => {
        // cast here because key types are collapsed on iteration
        const childText = groupInfo[key as unknown as 1 | 2 | 3].child;
        if (val === 'open') {
          expect(screen.getByText(childText)).toBeVisible();
        } else {
          expect(screen.getByText(childText)).not.toBeVisible();
        }
      });
    };
    it('renders the accordion group', async () => {
      groupSetup();
      expectGroupState({
        [1]: 'closed',
        [2]: 'closed',
        [3]: 'closed',
      });
      const oneTitle = groupInfo[1].title;
      await user.click(screen.getByText(oneTitle));
      expectGroupState({
        [1]: 'open',
        [2]: 'closed',
        [3]: 'closed',
      });
    });
    it('can be controlled via openKey', async () => {
      groupSetup({ openKey: 1 });
      const expectedGroupState = {
        [1]: 'open',
        [2]: 'closed',
        [3]: 'closed',
      } as const;
      expectGroupState(expectedGroupState);
      const oneTitle = groupInfo[1].title;
      await user.click(screen.getByText(oneTitle));
      expectGroupState(expectedGroupState);
    });
    it('can have defaults set via defaultOpenKey', async () => {
      groupSetup({ defaultOpenKey: 1 });
      expectGroupState({
        [1]: 'open',
        [2]: 'closed',
        [3]: 'closed',
      });
      const oneTitle = groupInfo[1].title;
      await user.click(screen.getByText(oneTitle));
      expectGroupState({
        [1]: 'closed',
        [2]: 'closed',
        [3]: 'closed',
      });
    });
    it('can set multiple children to open via openKey', () => {
      groupSetup({ openKey: [1, 3] });
      expectGroupState({
        [1]: 'open',
        [2]: 'closed',
        [3]: 'open',
      });
    });
    it('can default multiple children to open via defaultOpenKey', () => {
      groupSetup({ defaultOpenKey: [1, 3] });
      expectGroupState({
        [1]: 'open',
        [2]: 'closed',
        [3]: 'open',
      });
    });
    it('can keep only one child open in exclusive mode', async () => {
      groupSetup({ defaultOpenKey: 3, exclusive: true });
      expectGroupState({
        [1]: 'closed',
        [2]: 'closed',
        [3]: 'open',
      });
      const oneTitle = groupInfo[1].title;
      await user.click(screen.getByText(oneTitle));
      expectGroupState({
        [1]: 'open',
        [2]: 'closed',
        [3]: 'closed',
      });
    });
  });
});
