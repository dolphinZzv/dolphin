import AppKit

class BlockerView: NSView {
    override var acceptsFirstResponder: Bool { false }

    override init(frame: NSRect) {
        super.init(frame: frame)
        wantsLayer = true
        layer?.backgroundColor = NSColor.black.withAlphaComponent(0.08).cgColor
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func hitTest(_ point: NSPoint) -> NSView? { self }

    override func mouseDown(with event: NSEvent) {}
    override func mouseUp(with event: NSEvent) {}
    override func rightMouseDown(with event: NSEvent) {}
    override func rightMouseUp(with event: NSEvent) {}
    override func mouseMoved(with event: NSEvent) {}
    override func mouseDragged(with event: NSEvent) {}
    override func scrollWheel(with event: NSEvent) {}
    override func keyDown(with event: NSEvent) {}
    override func keyUp(with event: NSEvent) {}
}