using System;
using System.Drawing;
using System.Linq;
using System.Windows;
using System.Windows.Threading;
using Wf = System.Windows.Forms;

namespace Dolphin.WebHost
{
    public partial class App : Application
    {
        private McpServer? _server;
        private MainWindow? _mainWindow;
        private Wf.NotifyIcon? _trayIcon;

        protected override void OnStartup(StartupEventArgs e)
        {
            base.OnStartup(e);

            _mainWindow = new MainWindow();

            try
            {
                _server = new McpServer(port: 9223);
                _server.Start();
                _mainWindow.DataContext = _server;
                _mainWindow.SetStatus($"WebHost running on http://localhost:9223");
            }
            catch (Exception ex)
            {
                _mainWindow.SetStatus($"Failed to start: {ex.Message}");
                MessageBox.Show($"Failed to start WebHost server: {ex.Message}",
                    "Error", MessageBoxButton.OK, MessageBoxImage.Error);
            }

            SetupTrayIcon();
            DispatcherUnhandledException += OnDispatcherUnhandledException;
        }

        private void SetupTrayIcon()
        {
            _trayIcon = new Wf.NotifyIcon
            {
                Icon = SystemIcons.Application,
                Text = "Dolphin WebHost",
                Visible = true
            };

            var menu = new Wf.ContextMenuStrip();
            menu.Items.Add("Show Browser", null, (_, _) => ShowBrowser());
            menu.Items.Add("Dashboard", null, (_, _) => ShowDashboard());
            menu.Items.Add(new Wf.ToolStripSeparator());
            menu.Items.Add("Exit", null, (_, _) => OnExitFromTray());
            _trayIcon.ContextMenuStrip = menu;
            _trayIcon.DoubleClick += (_, _) => ShowBrowser();
        }

        private void ShowBrowser()
        {
            if (_server == null) return;
            var sessions = _server.SessionManager.ListSessions();
            var last = sessions.OrderByDescending(s => s.LastActivityAt).FirstOrDefault();
            if (last == null) return;

            var session = _server.SessionManager.GetSession(last.SessionId);
            if (session == null) return;

            _ = session.SetInteractiveAsync(true);
        }

        private void ShowDashboard()
        {
            if (_mainWindow == null) return;
            _mainWindow.Show();
            _mainWindow.WindowState = WindowState.Normal;
            _mainWindow.Activate();
        }

        private void OnExitFromTray()
        {
            Shutdown();
        }

        protected override void OnExit(ExitEventArgs e)
        {
            if (_trayIcon != null)
            {
                _trayIcon.Visible = false;
                _trayIcon.Dispose();
                _trayIcon = null;
            }
            _server?.Stop();
            _server?.Dispose();
            base.OnExit(e);
        }

        private void OnDispatcherUnhandledException(object sender,
            DispatcherUnhandledExceptionEventArgs e)
        {
            MessageBox.Show($"Unhandled error: {e.Exception.Message}",
                "Error", MessageBoxButton.OK, MessageBoxImage.Error);
            e.Handled = true;
        }
    }
}
