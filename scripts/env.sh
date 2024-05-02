sudo apt-get install -y libopus-dev libopusfile-dev ffmpeg
echo "127.0.0.1 converter" | sudo tee -a /etc/hosts
echo "alias ffmpeg='ffmpeg -hide_banner'" | tee -a ~/.bashrc
