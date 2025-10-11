# Setting up Ambian on a pi

These are _raw_ notes. Instructions will need to be made friendly.

## Hardware requirements

- 32GB SD card
- SD to USB adapter

## Creating OS boot on SD

### References

- See [Armbian OS getting started](https://docs.armbian.com/User-Guide_Getting-Started/)
- See [Orange Pi Zero 3 setup](https://www.armbian.com/orange-pi-zero-3/)

### Steps

Prepare image:

- Download [OS image](https://www.armbian.com/download/).
- Install [usbimager](https://gitlab.com/bztsrc/usbimager).
- Insert SD card into USB adapter into ops computer.
- Launch `USB Imager` and use it to write your downloaded image to the SD card. It might take an hour.
- When written, copy infrastructure_setup/dnsmasq/.not_logged_in_yet to the sd's `root/` dir.

Boot:

- Put the SD card into the Pi and boot.
- Set config options.
- From your ops machine, ssh into the pi.
- Run `armbian-install`
  - Select the option to put the boot loader on the sdcard

Configure:

```bash
# Block all suspend, hibernate, and sleep actions.
sudo systemctl mask sleep.target suspend.target hibernate.target hybrid-sleep.target
```

Update:

```bash
wget -qO - https://apt.armbian.com/armbian.key | gpg --dearmor | sudo tee /usr/share/keyrings/armbian.gpg >/dev/null
sudo apt update
sudo apt upgrade
```
