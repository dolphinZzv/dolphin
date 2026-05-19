using System.Windows;
using System.Windows.Threading;

namespace Dolphin.WebHost
{
    public partial class MainWindow : Window
    {
        private DispatcherTimer? _refreshTimer;

        public MainWindow()
        {
            InitializeComponent();
            _refreshTimer = new DispatcherTimer
            {
                Interval = System.TimeSpan.FromSeconds(3)
            };
            _refreshTimer.Tick += (_, _) => RefreshSessionList();
            _refreshTimer.Start();
        }

        public void SetStatus(string status)
        {
            if (Dispatcher.CheckAccess())
            {
                StatusText.Text = status;
            }
            else
            {
                Dispatcher.Invoke(() => StatusText.Text = status);
            }
        }

        private void RefreshSessionList()
        {
            if (DataContext is McpServer server)
            {
                var sessions = server.SessionManager.ListSessions();
                SessionList.ItemsSource = sessions;
            }
        }

        private async void OnShowWindowClick(object sender, RoutedEventArgs e)
        {
            var selected = SessionList.SelectedItem as Models.SessionInfo;
            if (selected == null)
            {
                SetStatus("No session selected");
                return;
            }
            if (!(DataContext is McpServer server)) return;
            var session = server.SessionManager.GetSession(selected.SessionId);
            if (session == null)
            {
                SetStatus("Session not found");
                return;
            }
            await session.SetInteractiveAsync(true);
            SetStatus($"Browser shown: {selected.SessionId}");
        }

        private void OnExitClick(object sender, RoutedEventArgs e)
        {
            Application.Current.Shutdown();
        }
    }
}
