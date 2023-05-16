import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

class FsPage extends StatelessWidget {
  const FsPage({super.key, required this.subpath});

  final String subpath;

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<_FsData>>(
      future: fetchData(subpath),
      builder: (context, snapshot) {
        if (snapshot.hasData) {
          final data = snapshot.data!;
          return ListView.builder(
            itemCount: data.length,
            itemBuilder: (context, index) {
              return ListTile(
                title: Text(data[index].label),
              );
            },
          );
        } else if (snapshot.hasError) {
          return Text("${snapshot.error}");
        }

        return const CircularProgressIndicator();
      },
    );

    throw UnimplementedError();
  }
}

class _FsData {
  final bool isDir;
  final String imageUrl;
  final String label;
  final String subpath;

  _FsData({
    required this.isDir,
    required this.imageUrl,
    required this.label,
    required this.subpath,
  });

  factory _FsData.fromJson(Map<String, dynamic> json) {
    return _FsData(
      isDir: json['IsDir'] ?? false,
      imageUrl: json['ImageUrl'] ?? '',
      label: json['Label'] ?? '',
      subpath: json['Subpath'] ?? '',
    );
  }
}

Future<List<_FsData>> fetchData(String subpath) async {
  final response = await http.get(Uri.parse('./$subpath'));

  if (response.statusCode == 200) {
    final List<dynamic> jsonData = jsonDecode(response.body);
    return jsonData.map((item) => _FsData.fromJson(item)).toList();
  } else {
    throw Exception('Failed to fetch data');
  }
}
