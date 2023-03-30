const { default: router } = await vi.importActual<typeof import('router')>('router');

router.initRouter(<div />);
export default router;
