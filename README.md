Gopherseerr - A Simple Radarr & Sonarr Request App
Gopherseerr is a lightweight, self-hosted web application that provides a clean and simple interface for users to search for movies and TV shows and add them to your Radarr and Sonarr libraries. It's built with Go and designed to be a single, easy-to-run binary.
The project was born out of a frustration with Overseerr's Docker requirement, offering a simpler, native alternative for users, particularly on Windows.

This app is perfect for giving friends or family a simple, mobile-friendly way to request content without giving them full access to your Sonarr/Radarr instances.

Key Features
Unified Search: A single search bar for both movies and TV shows, powered by the TMDB API.
Radarr Integration: Add movie requests directly to your Radarr library.
Granular Sonarr Control: When adding a TV show, you can choose to download:
The entire show (all seasons).
A specific season.
A single, individual episode.
Simple & Clean UI: A responsive, mobile-friendly interface designed for ease of use.
Easy Configuration: All settings are managed in a single config.json file.
Lightweight Deployment: Compiles to a single binary with no external dependencies needed at runtime (apart from the config and templates).
Runs in the Background: Can be compiled to run as a hidden background process or a full Windows Service.
Screenshots

Search Page:
![image](https://github.com/user-attachments/assets/ee1fbf78-318f-425c-bf35-0486d6fbb37d)

Results Page:
![image](https://github.com/user-attachments/assets/ad7956ac-8225-41eb-b92f-a2692b6c3a16)

Show Details Page:
![image](https://github.com/user-attachments/assets/c9b98581-801c-4ea7-b980-cb1392119bbc)

Requirements
Go: Version 1.21 or newer (only needed if you are compiling from source).
Radarr: A running instance of Radarr v4+.
Sonarr: A running instance of Sonarr v3+.
TMDB API Key: A free API key from The Movie Database (TMDB).
Installation & Setup
Clone or Download
Clone this repository to your machine or download the source code as a ZIP file.

Bash

git clone https://github.com/your-username/gopherseerr.git
cd gopherseerr
Create Configuration File
Create a new file named config.json in the root directory and paste the following content into it.

JSON

{
  "port": "8080",
  "tmdb_api_key": "",
  "radarr_url": "http://localhost:7878",
  "radarr_api_key": "",
  "sonarr_url": "http://localhost:8989",
  "sonarr_api_key": "",
  "radarr_root_folder": "",
  "sonarr_root_folder": ""
}
Edit config.json
Fill in the details for your setup:

tmdb_api_key: Your API key from TMDB.
radarr_url / sonarr_url: The URL to access your Radarr and Sonarr instances.
radarr_api_key / sonarr_api_key: Find these in Sonarr/Radarr under Settings -> General -> Security.
radarr_root_folder / sonarr_root_folder: The root path where your media is stored.
Find this in Radarr/Sonarr under Settings -> Media Management -> Root Folders.
Important for Windows users: Use double backslashes (\\) for paths in JSON, for example: "C:\\Media\\Movies".
Run the Application
Open a terminal or command prompt in the project directory and run:

Bash

go run .
The server should now be running!

Usage
Open your web browser and navigate to http://localhost:8080 (or whichever port you specified).
Use the search bar to find a movie or TV show.
From the results, you can request a movie directly or click "View Details" for a TV show to select specific seasons or episodes.
Compiling for Production (Windows)
To create a standalone executable that you can run anywhere, follow these steps.

Build for Background Execution
To create an .exe that runs silently in the background without a black command window, use the following build command:

DOS

go build -ldflags="-H=windowsgui"
Deploy the Files
Create a permanent folder for your app (e.g., C:\Gopherseerr). Copy the following three items into this folder:

The compiled gopherseerr.exe file.
Your completed config.json file.
The entire templates directory.
Running on Windows Startup
To have the app run automatically and silently when your PC starts:

Follow the "Compiling for Production" steps above.
Create a shortcut to the gopherseerr.exe file.
Press Windows Key + R, type shell:startup, and press Enter.
Move the shortcut you created into this Startup folder.
For a more robust setup, consider using a tool like NSSM to run the executable as a true Windows Service.
